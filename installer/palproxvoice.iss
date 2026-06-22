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
SelectDirDesc=Escolha a pasta do Palworld.
SelectDirLabel3=Auto-detectamos o Palworld do Steam. Se nao achou, clique em Procurar e escolha a pasta do Palworld (ou uma pasta proxima) — o instalador procura o jogo dentro dela.

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
const
  ATTR_DIR = $10; // FILE_ATTRIBUTE_DIRECTORY

var
  PalRoot: String; // raiz do Palworld resolvida em runtime; usada por GetBinDir

// uma pasta e raiz do Palworld se tiver Pal\Binaries dentro
function HasBinaries(dir: String): Boolean;
begin
  Result := (dir <> '') and DirExists(dir + '\Pal\Binaries');
end;

// sobe um nivel no caminho ('' quando chega na raiz do drive)
function ParentDir(dir: String): String;
var
  p: Integer;
begin
  Result := dir;
  while (Length(Result) > 0) and (Result[Length(Result)] = '\') do
    Delete(Result, Length(Result), 1);
  p := Length(Result);
  while (p > 0) and (Result[p] <> '\') do
    Dec(p);
  if p > 0 then
    Result := Copy(Result, 1, p - 1)
  else
    Result := '';
end;

// procura pra baixo (limitado) por <X>\Pal\Binaries dentro de 'dir'
function SearchDown(dir: String; depth: Integer): String;
var
  fr: TFindRec;
begin
  Result := '';
  if HasBinaries(dir) then
  begin
    Result := dir;
    exit;
  end;
  if depth <= 0 then
    exit;
  if FindFirst(dir + '\*', fr) then
  try
    repeat
      if ((fr.Attributes and ATTR_DIR) <> 0) and (fr.Name <> '.') and (fr.Name <> '..') then
      begin
        Result := SearchDown(dir + '\' + fr.Name, depth - 1);
        if Result <> '' then
          exit;
      end;
    until not FindNext(fr);
  finally
    FindClose(fr);
  end;
end;

// dado o que o usuario escolheu, acha a RAIZ do Palworld (pasta com Pal\Binaries):
// 1) a propria pasta ou um ancestral (escolheu fundo demais, ex ...\Pal\Binaries\Win64)
// 2) procura pra baixo (escolheu uma pasta-mae, ex a 'common' do Steam)
function ResolvePalRoot(dir: String): String;
var
  cur: String;
  i: Integer;
begin
  Result := '';
  cur := dir;
  for i := 0 to 4 do
  begin
    if HasBinaries(cur) then
    begin
      Result := cur;
      exit;
    end;
    cur := ParentDir(cur);
    if cur = '' then
      break;
  end;
  Result := SearchDown(dir, 3);
end;

// raiz do Palworld do Steam (biblioteca principal + demais libraryfolders.vdf)
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
  if HasBinaries(steam + '\steamapps\common\Palworld') then
  begin
    Result := steam + '\steamapps\common\Palworld';
    exit;
  end;
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
            if HasBinaries(lib + '\steamapps\common\Palworld') then
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
var
  root: String;
begin
  if PalRoot <> '' then
    root := PalRoot
  else
    root := ExpandConstant('{app}');
  if DirExists(root + '\Pal\Binaries\WinGDK') then
    Result := root + '\Pal\Binaries\WinGDK'
  else
    Result := root + '\Pal\Binaries\Win64';
end;

// resolve a raiz a partir da pasta escolhida; erra so se nao achar Palworld por perto
function NextButtonClick(CurPageID: Integer): Boolean;
var
  root: String;
begin
  Result := True;
  if CurPageID = wpSelectDir then
  begin
    root := ResolvePalRoot(ExpandConstant('{app}'));
    if root = '' then
    begin
      MsgBox('Nao achei o Palworld dentro dessa pasta. Escolha a pasta do Palworld (ou uma pasta acima dela).', mbError, MB_OK);
      Result := False;
    end
    else
      PalRoot := root; // guarda a raiz resolvida pro GetBinDir
  end;
end;
