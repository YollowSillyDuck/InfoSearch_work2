# InteractiveSearch.ps1

param (
    [string]$ServerUrl = "http://127.0.0.1:8080"
)

$Query = Read-Host -Prompt "请输入你要搜索的关键词 (直接回车默认搜索 peripheral)"
if ([string]::IsNullOrWhiteSpace($Query)) {
    $Query = "peripheral"
}

Write-Host "`n正在检索关键词: '$Query' ..." -ForegroundColor Cyan
$searchUrl = "$ServerUrl/search/documents?query=$Query"
$response = Invoke-RestMethod -Uri $searchUrl

if ($null -eq $response.data -or $response.data.Count -eq 0) {
    Write-Host "未找到相关文档。" -ForegroundColor Yellow
    exit
}

Write-Host "`n================ 搜索结果 ================" -ForegroundColor Green
$response.data | Format-List id, title, relevance, main_match

Write-Host "`n================ 人工评价 ================" -ForegroundColor Cyan
Write-Host "请根据上方打印的内容，对以下文档进行打分：" -ForegroundColor DarkGray

foreach ($doc in $response.data) {
    $promptMsg = "文档 ID: $($doc.id) | 标题: $($doc.title)`n这个结果准确吗？(y=准确 / n=不准确 / s=跳过): "
    $answer = Read-Host -Prompt $promptMsg

    $isRelevant = $null
    if ($answer -eq 'y' -or $answer -eq 'Y') {
        $isRelevant = $true
    }
    elseif ($answer -eq 'n' -or $answer -eq 'N') {
        $isRelevant = $false
    }

    if ($null -ne $isRelevant) {
        $evalBody = @{
            query       = $Query
            document_id = $doc.id
            is_relevant = $isRelevant
        } | ConvertTo-Json

        $evalUrl = "$ServerUrl/search/evaluate"
        Invoke-RestMethod -Uri $evalUrl -Method Post -ContentType "application/json" -Body $evalBody | Out-Null
        Write-Host "-> 评价已记录！`n" -ForegroundColor Green
    }
    else {
        Write-Host "-> 已跳过。`n" -ForegroundColor DarkGray
    }
}

Write-Host "所有评价完成！当前系统准确率统计如下：" -ForegroundColor Cyan
Invoke-RestMethod -Uri "$ServerUrl/search/metrics" | ConvertTo-Json -Depth 5