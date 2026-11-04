"""Install and exercise the native package on disposable CI runners.

Only matching architectures execute; other targets remain package-build checks.
All board content and credentials are isolated test data.
"""
import json
import argparse
import os
import pathlib
import platform
import socket
import subprocess
import tempfile
import time
import uuid
import urllib.error
import urllib.request

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("target")
parser.add_argument("arch")
parser.add_argument("--binary", type=pathlib.Path, help="Test a supplied local build without installing a package; CI omits this option.")
options = parser.parse_args()
target, arch = options.target, options.arch
binary_kind = "supplied" if options.binary else "installed"
native = {"x86_64": "amd64", "amd64": "amd64", "aarch64": "arm64", "arm64": "arm64"}.get(platform.machine().lower())
if native != arch:
    print(f"Built {target}/{arch}; native execution requires a matching runner (this runner is {native}).")
    raise SystemExit(0)

root = pathlib.Path.cwd()
def run(*args, env=None):
    return subprocess.run([str(a) for a in args], check=True, env=env, capture_output=True, text=True)

with tempfile.TemporaryDirectory(prefix="player2-installed-smoke-") as tmp:
    temp = pathlib.Path(tmp)
    env = dict(os.environ, XDG_CONFIG_HOME=str(temp / "config"), APPDATA=str(temp / "config"), GITHUB_TOKEN="")
    if options.binary:
        binary = options.binary.resolve(strict=True)
        print("Testing a supplied local binary. Native installation is NOT tested in this mode.")
    elif target == "windows":
        install = temp / "installed"
        run(root / f"desktop/Player2-Setup-windows-{arch}.exe", "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", f"/DIR={install}", env=env)
        binary = install / "player2-kanban.exe"
        assert (install / "Player2.exe").is_file(), "Desktop launcher missing"
    elif target == "darwin":
        run("sudo", "installer", "-pkg", root / f"desktop/Player2-mac-{arch}.pkg", "-target", "/")
        binary = pathlib.Path("/Applications/Player2.app/Contents/MacOS/Player2")
        assert pathlib.Path("/usr/local/bin/player2-kanban").is_file()
    elif target == "linux":
        run("sudo", "apt-get", "update", "-qq")
        run("sudo", "apt-get", "install", "-y", str(root / f"desktop/Player2-linux-{arch}.deb"))
        binary = pathlib.Path("/usr/bin/player2-kanban")
        assert pathlib.Path("/usr/share/applications/player2.desktop").is_file()
    else:
        raise SystemExit("Unsupported platform")

    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        port = sock.getsockname()[1]
    database = temp / "boards.db"
    base = f"http://127.0.0.1:{port}"
    def api(path, method="GET", payload=None):
        data = None if payload is None else json.dumps(payload).encode()
        request = urllib.request.Request(base + path, data=data, method=method, headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(request, timeout=10) as response:
            return json.load(response) if response.status != 204 else None

    # Exercise installed open/reuse, not a separately built executable.
    try:
        run(binary, "--db", database, "open", "--port", port, "--no-browser", env=env)
        run(binary, "--db", database, "open", "--port", port, "--no-browser", env=env)
        connection = json.loads(run(binary, "--db", database, "status", "--port", port, env=env).stdout)
        assert connection["serverRunning"] and connection["databaseMatches"], "Installed app could not identify its board"
        other_database = temp / "another-board.db"
        mismatch = json.loads(run(binary, "--db", other_database, "status", "--port", port, env=env).stdout)
        assert not mismatch["serverRunning"] and not mismatch["databaseMatches"], "Another database was reported as connected"
        refused = subprocess.run([str(binary), "--db", str(other_database), "open", "--port", str(port), "--no-browser"], env=env, capture_output=True, text=True, timeout=15)
        assert refused.returncode != 0, "Opening another database silently reused the visible board"
        project = api("/api/projects", "POST", {"name": "Installer smoke", "prefix": "SMK", "completionPolicy": "manual"})
        task = api("/api/tickets", "POST", {"projectId": project["id"], "title": "Plan the week", "dueDate": "2026-10-23", "repeatRule": "weekly"})
        completed = api(f'/api/tickets/{task["id"]}/move', "POST", {"status": "done", "expectedRevision": task["revision"]})
        assert len(api("/api/tickets")) == 2, "Recurring task missing"
        api(f'/api/tickets/{task["id"]}?revision={completed["revision"]}', "DELETE")
        deleted = api("/api/trash")[0]
        api(f'/api/trash/{deleted["id"]}/restore', "POST", {"revision": deleted["revision"]})
        backup = api("/api/recovery/backups", "POST")
        preview = api(f'/api/recovery/backups/{backup["id"]}/preview', "POST")
        first = api("/api/recovery/restore", "POST", {"id": preview["id"], "confirm": True})
        assert api("/api/recovery/restore", "POST", {"id": preview["id"], "confirm": True}) == first
        assert len(api("/api/projects")) == 2
        # Drive the real installed stdio server, rather than pretending a chat
        # response or a copied setup command proves that an agent connected.
        pilot_id = "connect-" + str(uuid.uuid4())
        pilot = api("/api/connections/pilots", "POST", {"requestId": pilot_id, "goal": "Prepare a short weekly plan"})
        again = api("/api/connections/pilots", "POST", {"requestId": pilot_id, "goal": "Prepare a short weekly plan"})
        assert again["task"]["id"] == pilot["task"]["id"] and not pilot["reported"]
        plan = api("/api/connections/setup")
        configured = plan["config"]["mcpServers"][plan["name"]]
        def mcp(name, arguments):
            messages = [
                {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "installer-test", "version": "1"}}},
                {"jsonrpc": "2.0", "method": "notifications/initialized"},
                {"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": name, "arguments": arguments}},
            ]
            response = subprocess.run([configured["command"], *configured["args"]], input="".join(json.dumps(m) + "\n" for m in messages), capture_output=True, text=True, env=env, check=True, timeout=20)
            parsed = [json.loads(line) for line in response.stdout.splitlines()]
            result = next(item for item in parsed if item.get("id") == 2)
            assert "error" not in result and not result["result"].get("isError"), result
            return json.loads(result["result"]["content"][0]["text"])
        task_id = pilot["task"]["id"]
        current = mcp("get_ticket", {"id": task_id})
        mcp("report_activity", {"id": task_id, "agent": "Automated protocol test", "sessionId": pilot_id, "summary": "Planning"})
        assert api(f"/api/connections/pilots/{pilot_id}")["reported"]
        current = mcp("move_ticket", {"id": task_id, "status": "in_progress", "expectedRevision": current["revision"]})
        current = mcp("update_ticket", {"id": task_id, "description": "Prepare a short weekly plan\n1. List commitments\n2. Choose three priorities\n3. Reserve time", "expectedRevision": current["revision"]})
        mcp("report_activity", {"id": task_id, "agent": "Automated protocol test", "sessionId": pilot_id, "summary": "Plan saved for review"})
        current = mcp("move_ticket", {"id": task_id, "status": "review", "expectedRevision": current["revision"]})
        assert api(f"/api/connections/pilots/{pilot_id}")["state"] == "needs_review"
        api(f"/api/tickets/{task_id}/move", "POST", {"status": "done", "expectedRevision": current["revision"]})
        assert api(f"/api/connections/pilots/{pilot_id}")["state"] == "completed"
        print(f"PASS: guided first-task creation/retry, {binary_kind} stdio MCP reports and result, HTTP review completion. This is protocol automation, not a human or vendor-agent pilot.")
        with urllib.request.urlopen(base + "/recovery", timeout=10) as response:
            assert response.status == 200 and "script-src 'self'" in response.headers["Content-Security-Policy"]
        print(f"PASS: {binary_kind} app open/reuse, matching database discovery, mismatch refusal, recurrence, delete/undo, backup/restore, retry, embedded UI.")
    finally:
        subprocess.run([str(binary), "--db", str(database), "stop"], env=env, capture_output=True, timeout=15)
        for _ in range(50):
            try:
                urllib.request.urlopen(base + "/api/health", timeout=.2).close()
            except (OSError, urllib.error.URLError):
                break
            time.sleep(.1)
    assert database.exists(), "Stopping removed user data"
    if target == "windows" and not options.binary:
        run(install / "unins000.exe", "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", env=env)
        assert database.exists(), "Uninstall removed boards"
        print("PASS: Windows uninstall preserves boards.")
