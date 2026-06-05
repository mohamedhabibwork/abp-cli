Write-Host "Scanning solutions..." -ForegroundColor Cyan

# -------------------------------
# Infra detection rules
# -------------------------------
$InfraRules = @{
    "SQL Server" = @(
        "Microsoft.Data.SqlClient",
        "System.Data.SqlClient",
        "EntityFramework",
        "Dapper"
    )
    "Oracle Client" = @(
        "Oracle.ManagedDataAccess"
    )
    "PostgreSQL" = @(
        "Npgsql"
    )
    "Redis" = @(
        "StackExchange.Redis",
        "Microsoft.Extensions.Caching.Redis"
    )
    "RabbitMQ" = @(
        "RabbitMQ.Client",
        "MassTransit.RabbitMQ"
    )
    "Hangfire" = @(
        "Hangfire"
    )
    "Elasticsearch" = @(
        "NEST",
        "Elasticsearch.Net"
    )
}

# -------------------------------
# Discover solutions
# -------------------------------
$solutions = Get-ChildItem -Recurse -Filter *.sln
if (-not $solutions) {
    Write-Error "No .sln files found."
    return
}

# Global infra tracker
$globalInfra = @{}
foreach ($key in $InfraRules.Keys) { $globalInfra[$key] = $false }

$output = @()
$output += "# Application Server Requirements – Microservices"
$output += ""
$output += "_Generated on $(Get-Date -Format 'yyyy-MM-dd HH:mm')_"
$output += ""

# -------------------------------
# Helper: Detect project type
# -------------------------------
function Get-ProjectType {
    param ($csproj)

    if (-not $csproj -or -not $csproj.FullName) { return "Unknown (Skipped)" }
    if (-not (Test-Path $csproj.FullName)) { return "Unknown (File Missing)" }

    $content = Get-Content $csproj.FullName -Raw
    if ($content -match "Microsoft.NET.Sdk.Web") { return "ASP.NET Core Web API (IIS)" }
    elseif ($content -match "Worker") { return "Worker Service (Windows Service)" }
    else { return "Class / Console" }
}

# -------------------------------
# Process each solution
# -------------------------------
foreach ($sln in $solutions) {

    $output += "## Service: $($sln.BaseName)"

    # Initialize service-level infra
    $serviceInfra = @{}
    foreach ($key in $InfraRules.Keys) { $serviceInfra[$key] = $false }

    # Initialize list of all detected packages/components
    $allPackages = @{}

    # Get valid csproj files
    $projects = Get-ChildItem $sln.Directory -Recurse -Filter *.csproj |
                Where-Object { $_.FullName -and (Test-Path $_.FullName) }

    foreach ($proj in $projects) {

        # Collect packages
        $xml = [xml](Get-Content $proj.FullName)
        $packages = @()
        foreach ($pkg in $xml.Project.ItemGroup.PackageReference) { 
            if ($pkg.Include -and $pkg.Include.Trim() -ne "") {
                $packages += $pkg.Include
            }
        }

        # Add all packages to service-wide list (skip null/empty)
        foreach ($p in $packages) {
            if ($p -and $p.Trim() -ne "") {
                $allPackages[$p] = $true
            }
        }


        # Run infra detection
        foreach ($infraName in $InfraRules.Keys) {
            foreach ($rule in $InfraRules[$infraName]) {
                if ($packages -match [regex]::Escape($rule)) {
                    $serviceInfra[$infraName] = $true
                    $globalInfra[$infraName] = $true
                }
            }
        }
    }

    # --- OUTPUT ---
    $output += "**Detected Components (all):**"
    foreach ($pkg in $allPackages.Keys | Sort-Object) {
        $output += "- $pkg"
    }

    # Optional: also show official infra
    $output += "**Required Components (InfraRules):**"
    foreach ($key in $serviceInfra.Keys) {
        if ($serviceInfra[$key]) { $output += "- $key" }
    }

    $output += ""  # blank line between services
}

# --- Global shared infra ---
$output += "## Shared Infrastructure Across All Services"
foreach ($key in $globalInfra.Keys) {
    if ($globalInfra[$key]) { $output += "- $key" }
}

# Write Markdown file
$output | Out-File "Microservices-AppServer-Requirements.md" -Encoding UTF8

Write-Host "✔ Microservices-AppServer-Requirements.md generated successfully" -ForegroundColor Green
