"""Package existing Go binaries using native OS tools. Run from the repo root."""
import json, os, pathlib, shutil, subprocess, sys, tempfile
platform, arch = sys.argv[1:]
root = pathlib.Path.cwd()
out = root / 'desktop'
out.mkdir(exist_ok=True)
version = json.loads((root/'package.json').read_text())['version']
binary = root / f'player2-kanban-{platform}-{arch}'
def run(*args):
    subprocess.run(list(map(str,args)), check=True)
with tempfile.TemporaryDirectory(prefix='player2-package-') as temp:
    stage=pathlib.Path(temp)
    if platform == 'darwin':
        app=stage/'Applications/Player2.app/Contents'
        (app/'MacOS').mkdir(parents=True)
        shutil.copy2(binary,app/'MacOS/Player2')
        (app/'MacOS/Player2').chmod(0o755)
        import plistlib
        (app/'Info.plist').write_bytes(plistlib.dumps({'CFBundleName':'Player2','CFBundleDisplayName':'Player2','CFBundleIdentifier':'com.player2.kanban','CFBundleExecutable':'Player2','CFBundlePackageType':'APPL','CFBundleShortVersionString':version,'CFBundleVersion':version.split('-')[0],'LSUIElement':True}))
        cli=stage/'usr/local/bin';cli.mkdir(parents=True)
        shutil.copy2(binary,cli/'player2-kanban');(cli/'player2-kanban').chmod(0o755)
        run('pkgbuild','--root',stage,'--identifier','com.player2.kanban','--version',version.split('-')[0],'--install-location','/',out/f'Player2-mac-{arch}.pkg')
    elif platform == 'linux':
        cli=stage/'usr/bin';cli.mkdir(parents=True)
        shutil.copy2(binary,cli/'player2-kanban');(cli/'player2-kanban').chmod(0o755)
        apps=stage/'usr/share/applications';apps.mkdir(parents=True)
        (apps/'player2.desktop').write_text('[Desktop Entry]\nType=Application\nName=Player2\nComment=Your plans, tasks, and AI work\nExec=player2-kanban open\nTerminal=false\nCategories=Office;ProjectManagement;\n')
        control=stage/'DEBIAN';control.mkdir()
        (control/'control').write_text(f'Package: player2-kanban\nVersion: {version.replace("-","~")}\nArchitecture: {arch}\nMaintainer: Player2 contributors\nDepends: xdg-utils, zenity\nDescription: Your plans, tasks, and AI work in a local browser app.\n')
        run('dpkg-deb','--root-owner-group','--build',stage,out/f'Player2-linux-{arch}.deb')
    elif platform == 'windows':
        compiler=pathlib.Path(os.environ.get('ProgramFiles(x86)',r'C:\Program Files (x86)'))/'Inno Setup 6/ISCC.exe'
        run(compiler,f'/DAppVersion={version}',f'/DTargetArch={arch}',f'/O{out}',root/'packaging/windows.iss')
    else:
        raise SystemExit('Unsupported platform')
