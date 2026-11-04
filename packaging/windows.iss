#ifndef AppVersion
#define AppVersion "2.0.0-beta.1"
#endif
#ifndef TargetArch
#define TargetArch "amd64"
#endif
[Setup]
AppId=Player2Kanban
AppName=Player2
AppVersion={#AppVersion}
DefaultDirName={localappdata}\Programs\Player2
DefaultGroupName=Player2
PrivilegesRequired=lowest
#if TargetArch == "arm64"
ArchitecturesAllowed=arm64
ArchitecturesInstallIn64BitMode=arm64
#else
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
#endif
DisableProgramGroupPage=yes
OutputDir=desktop
OutputBaseFilename=Player2-Setup-windows-{#TargetArch}
Compression=lzma2
SolidCompression=yes
UninstallDisplayName=Player2
CloseApplications=yes
[Files]
Source: "..\player2-kanban-windows-{#TargetArch}.exe"; DestDir: "{app}"; DestName: "player2-kanban.exe"; Flags: ignoreversion
Source: "..\Player2.exe"; DestDir: "{app}"; Flags: ignoreversion
[Icons]
Name: "{group}\Player2"; Filename: "{app}\Player2.exe"
Name: "{autodesktop}\Player2"; Filename: "{app}\Player2.exe"
[Run]
Filename: "{app}\Player2.exe"; Description: "Open Player2"; Flags: nowait postinstall skipifsilent
[UninstallRun]
Filename: "{app}\player2-kanban.exe"; Parameters: "stop"; Flags: runhidden; RunOnceId: "StopPlayer2"
