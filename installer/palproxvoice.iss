; PalProxVoice — instalador (Inno Setup)
; Gera PalProxVoice-Setup.exe: acha o Palworld, instala UE4SS+mod no jogo,
; instala o companion+config, e configura auto-start (oculto, sem aba na taskbar).

[Setup]
AppName=PalProxVoice
AppVersion=0.7
AppPublisher=Fernando Braga
DefaultDirName={code:DetectPalworld}
AppendDefaultDirName=no
DirExistsWarning=no
DisableProgramGroupPage=yes
PrivilegesRequired=admin
OutputBaseFilename=PalProxVoice-Setup
OutputDir=.
Compression=lzma2
SolidCompression=yes
WizardStyle=modern

[Languages]
Name: "br"; MessagesFile: "compiler:Languages\BrazilianPortuguese.isl"

[Messages]
SelectDirDesc=Escolha a instalacao do Palworld onde instalar.
SelectDirLabel3=Auto-detectamos o Palworld do Steam. Se voce tem mais de uma instalacao (outra pasta, build diferente), clique em Procurar e escolha qual.

[Files]
; companion + config -> LocalAppData do usuario
Source: "palproxvoice.exe"; DestDir: "{localappdata}\PalProxVoice"; Flags: ignoreversion
Source: "config.json"; DestDir: "{localappdata}\PalProxVoice"; Flags: onlyifdoesntexist
; UE4SS v3.0.1 + mod -> a pasta Binaries certa do jogo (Win64 ou WinGDK, em runtime)
Source: "ue4ss\*"; DestDir: "{code:GetBinDir}"; Flags: recursesubdirs createallsubdirs ignoreversion

[Registry]
; Windows abaixa todos os sons quando o mic ativa ("ducking" de comunicacao).
; 3 = "nao fazer nada". (pega de vez apos reiniciar/relogar)
Root: HKCU; Subkey: "Software\Microsoft\Multimedia\Audio"; ValueType: dword; ValueName: "UserDuckingPreference"; ValueData: "3"

[Icons]
; auto-start: sobe OCULTO (sem aba na barra de tarefas) com o Windows; vira overlay no jogo
Name: "{userstartup}\PalProxVoice"; Filename: "{localappdata}\PalProxVoice\palproxvoice.exe"; Parameters: "-min"; WorkingDir: "{localappdata}\PalProxVoice"

[Run]
Filename: "{localappdata}\PalProxVoice\palproxvoice.exe"; Description: "Abrir o PalProxVoice agora"; Flags: nowait postinstall skipifsilent

[Code]
function SteamPalworld(): String;
var
  steam, vdf, s, lib: String;
  lines: TArrayOfString;
  i, p, q: Integer;
begin
  Result := '';
  if not RegQueryStringValue(HKCU, 'Software\Valve\Steam', 'SteamPath', steam) then
    exit;
  StringChangeEx(steam, '/', '\', True);
  // biblioteca principal
  if DirExists(steam + '\steamapps\common\Palworld\Pal\Binaries') then
  begin
    Result := steam + '\steamapps\common\Palworld';
    exit;
  end;
  // demais bibliotecas (libraryfolders.vdf)
  vdf := steam + '\steamapps\libraryfolders.vdf';
  if LoadStringsFromFile(vdf, lines) then
    for i := 0 to GetArrayLength(lines) - 1 do
    begin
      s := lines[i];
      p := Pos('"path"', s);
      if p > 0 then
      begin
        s := Copy(s, p + 6, Length(s));
        p := Pos('"', s);
        if p > 0 then
        begin
          s := Copy(s, p + 1, Length(s));
          q := Pos('"', s);
          if q > 0 then
          begin
            lib := Copy(s, 1, q - 1);
            StringChangeEx(lib, '\\', '\', True);
            if DirExists(lib + '\steamapps\common\Palworld\Pal\Binaries') then
            begin
              Result := lib + '\steamapps\common\Palworld';
              exit;
            end;
          end;
        end;
      end;
    end;
end;

function DetectPalworld(Param: String): String;
begin
  Result := SteamPalworld();
end;

function GetBinDir(Param: String): String;
begin
  if DirExists(ExpandConstant('{app}') + '\Pal\Binaries\WinGDK') then
    Result := ExpandConstant('{app}') + '\Pal\Binaries\WinGDK'
  else
    Result := ExpandConstant('{app}') + '\Pal\Binaries\Win64';
end;

// valida que a pasta escolhida e o Palworld
function NextButtonClick(CurPageID: Integer): Boolean;
begin
  Result := True;
  if CurPageID = wpSelectDir then
    if not DirExists(ExpandConstant('{app}') + '\Pal\Binaries') then
    begin
      MsgBox('Essa pasta nao parece ser o Palworld (falta Pal\Binaries). Escolha a pasta raiz do Palworld.', mbError, MB_OK);
      Result := False;
    end;
end;
