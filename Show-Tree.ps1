#function Show-Tree {
#    param (
#        [string]$Path = ".",
#        [int]$Level = 0,
#        [string[]]$Exclude = @()
#    )
#
#    # 获取当前目录下的子目录和文件
#    $items = Get-ChildItem -LiteralPath $Path -Force -ErrorAction SilentlyContinue |
#            Where-Object { $_.PSIsContainer -or $_.PSIsContainer -eq $false }
#
#    foreach ($item in $items) {
#        # 判断是否排除
#        if ($Exclude -contains $item.Name) {
#            continue
#        }
#
#        # 缩进 + 符号
#        $indent = (" " * ($Level * 2)) + "|- "
#        Write-Output "$indent$item"
#
#        # 如果是文件夹，递归调用
#        if ($item.PSIsContainer) {
#            Show-Tree -Path $item.FullName -Level ($Level + 1) -Exclude $Exclude
#        }
#    }
#}

function Show-Tree {
    param (
        [string]$Path = ".",
        [int]$Level = 0,
        [string[]]$Exclude = @()
    )

    $items = Get-ChildItem -LiteralPath $Path -Force -ErrorAction SilentlyContinue

    foreach ($item in $items) {
        # 只判断根目录下的直接子项是否在排除列表中
        if ($Exclude -contains $item.Name) {
            continue
        }

        # 打印树形结构前缀
        $indent = (" " * ($Level * 2)) + "|- "
        Write-Output "$indent$($item.Name)"

        # 递归进入子目录
        if ($item.PSIsContainer -and -not ($Exclude -contains $item.Name)) {
            Show-Tree -Path $item.FullName -Level ($Level + 1) -Exclude $Exclude
        }
    }
}

Show-Tree -Path "." -Exclude @(".idea", ".git", ".github", "deploy", "docker", "logs", "logs", "ui", "Show-Tree.ps1", "docker-compose.yml")
