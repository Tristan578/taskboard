"""Install and exercise the native package on disposable CI runners.

Only matching architectures execute; other targets remain package-build checks.
All board content and credentials are isolated test data.
"""
import json
import os
import pathlib
import platform
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request

target, arch = sys.argv[1:]
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
    if target == "windows":
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
        with urllib.request.urlopen(base + "/recovery", timeout=10) as response:
            assert response.status == 200 and "script-src 'self'" in response.headers["Content-Security-Policy"]
        print("PASS: installed app open/reuse, recurrence, delete/undo, backup/restore, retry, embedded UI.")
    finally:
        subprocess.run([str(binary), "--db", str(database), "stop"], env=env, capture_output=True, timeout=15)
        for _ in range(50):
            try:
                urllib.request.urlopen(base + "/api/health", timeout=.2).close()
            except (OSError, urllib.error.URLError):
                break
            time.sleep(.1)
    assert database.exists(), "Stopping removed user data"
    if target == "windows":
        run(install / "unins000.exe", "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", env=env)
        assert database.exists(), "Uninstall removed boards"
        print("PASS: Windows uninstall preserves boards.")
