Set shell = CreateObject("Shell.Application")
Set fso = CreateObject("Scripting.FileSystemObject")
exePath = fso.BuildPath(fso.GetParentFolderName(WScript.ScriptFullName), "PortManager.exe")

If fso.FileExists(exePath) Then
    shell.ShellExecute exePath, "", "", "runas", 1
Else
    MsgBox "未找到 PortManager.exe，请先编译程序。", 16, "PortManager"
End If
