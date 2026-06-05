<#
.SYNOPSIS
    Generates a communication matrix from appsettings.json files
.DESCRIPTION
    Scans all projects' appsettings*.json files, detects service-to-service URLs,
    third-party endpoints, and outputs a CSV with Service From, Service To, Communication Type, and Detail.
#>

Write-Host "Scanning appsettings files..." -ForegroundColor Cyan

# -------------------------------
# Helper: Detect local URLs
# -------------------------------
function IsLocalUrl {
    param($url)
    return $url -match 'localhost' -or $url -match '127\.0\.0\.1'
}

# -------------------------------
# Helper: Flatten JSON recursively
# -------------------------------
function Flatten-Json {
    param(
        [Parameter(Mandatory)]
        $json,
        [string]$prefix = ""
    )

    $flat = @{}

    if ($json -is [System.Collections.IDictionary] -or $json -is [pscustomobject]) {
        foreach ($key in $json.PSObject.Properties.Name) {
            $value = $json.$key
            $flat += Flatten-Json $value ($prefix + $key + ".")
        }
    }
    elseif ($json -is [System.Collections.IEnumerable] -and -not ($json -is [string])) {
        $i = 0
        foreach ($item in $json) {
            $flat += Flatten-Json $item ($prefix + $i + ".")
            $i++
        }
    }
    else {
        # For strings, numbers, booleans, etc.
        $flat[$prefix.TrimEnd(".")] = $json
    }

    return $flat
}


# -------------------------------
# Endpoint detection keywords
# -------------------------------
$endpointKeys = @(
    'BaseUrl','Authority','RootUrl','Endpoint','RedirectUri','Url'
)

# -------------------------------
# Find all appsettings*.json files
# -------------------------------
$appSettingsFiles = Get-ChildItem -Recurse -Filter "appsettings*.json"

if (-not $appSettingsFiles) {
    Write-Error "No appsettings*.json files found."
    return
}

$matrix = @()

foreach ($file in $appSettingsFiles) {
    Write-Host "Processing $($file.FullName)" -ForegroundColor Yellow

    try {
        $jsonRaw = Get-Content $file.FullName -Raw
        $jsonData = $jsonRaw | ConvertFrom-Json
    }
    catch {
        Write-Warning "Failed to parse $($file.FullName): $_"
        continue
    }

    $flatJson = Flatten-Json $jsonData

    $serviceName = $file.Directory.Name

    foreach ($key in $flatJson.Keys) {
        $value = $flatJson[$key]

        if ($value -is [string]) {
            $matchesKey = $false
            foreach ($ek in $endpointKeys) {
                if ($key -match $ek) {
                    $matchesKey = $true
                    break
                }
            }

            if ($matchesKey -and -not (IsLocalUrl $value)) {
                $matrix += [PSCustomObject]@{
                    'Service From' = $serviceName
                    'Service To'   = $key
                    'Communication Type' = 'HTTP/HTTPS'
                    'Detail'       = $value
                }
            }
        }
    }
}

# -------------------------------
# Export CSV
# -------------------------------
$outputFile = "Communication-Matrix.csv"
$matrix | Sort-Object 'Service From','Service To' | Export-Csv $outputFile -NoTypeInformation -Encoding UTF8

Write-Host "✔ Communication matrix generated: $outputFile" -ForegroundColor Green
