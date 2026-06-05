# CompleteAbpEntityGenerator.ps1
# Usage: .\CompleteAbpEntityGenerator.ps1 -EntityName "Product" -EntityType "FullAuditedAggregateRoot<Guid>" -Attributes @("string Name required maxlen:100", "decimal Price required", "int Stock")
param (  
    [string]$EntityName,
    [string]$EntityType, #Entity<Guid>|AggregateRoot<Guid>|ValueObject|FullAuditedAggregateRoot<Guid>
    [array]$Attributes

)
$solution = "EdeServices"
$Service = "EdeServices"
$folder= (Resolve-Path (Join-Path $PSScriptRoot "..\src")).Path


if ($EntityType -eq "") {$EntityType="FullAuditedAggregateRoot<Guid>"}
# Helper functions
function Get-Plural {
    param (
        [string]$Word
    )

    # Handle special cases
    if ($Word -match '(ox)$') {
        return "$Worden"
    }
    elseif ($Word -match '(ch|sh|ss|x|z)$') {
        return "$Wordes"
    }
    elseif ($Word -match '([^aeiouy])y$') {
        return ($Word -replace 'y$', 'ies')
    }
    elseif ($Word -match '(fe|f)$') {
        return ($Word -replace '(fe|f)$', 'ves')
    }
    elseif ($Word -match 'us$') {
        return ($Word -replace 'us$', 'i')
    }
    elseif ($Word -match 'is$') {
        return ($Word -replace 'is$', 'es')
    }
    return "${Word}s"
}
function ConvertFirstLetterToLowerCase {
    param (
        [string]$str
    )
    return $str.Substring(0,1).ToLower() + $str.Substring(1)
}
function GetRequiredAttributes{
    param (
        [array]$Attributes
    )
    $reqAttributes = ""
    foreach ($attribute in $Attributes) {
        $type, $name, $modifiers = $attribute -split ' ',3

        if ($modifiers -match "required") {
            if($reqAttributes.Length -gt 0) 
            {
                $reqAttributes += ","
            }
            $reqAttributes += "${type} $(ConvertFirstLetterToLowerCase $name)"
        }
    }
    return $reqAttributes
}
function GetOriginalAttributes{
    param (
        [array]$Attributes
    )
    $orgAttributes = ""
    foreach ($attribute in $Attributes) {
        $type, $name, $modifiers = $attribute -split ' ',3

        if ($modifiers -notmatch "ForeignKey") {
            if($orgAttributes.Length -gt 0) 
            {
                $orgAttributes += ","
            }
            $orgAttributes += "${type} $(ConvertFirstLetterToLowerCase $name)"
        }
    }
    return $orgAttributes
}
function GetAnnotations {
    param (
        [string]$definition
    )

    $type, $name, $modifiers = $definition -split ' ',3
    $annotations = ""

    if ($modifiers -match "required") {
        $annotations += "`n        [Required]"
    }

    if ($modifiers -match "maxlen:(\d+)") {
        $maxLength = $matches[1]
        $annotations += "`n        [MaxLength(${EntityName}Constants.ValidationConstants.${name}MaxLength)]"
    }

    return $annotations
}
function GetForeignKey {
    param (
        [string]$definition
    )

    $type, $name, $modifiers = $definition -split ' ',3
    $foreignKey = ""

    if ($modifiers -match "ForeignKey") {
        $foreignKey += @"
        `n        [ForeignKey("${name}Id")]
"@
    }

    return $foreignKey
}
function GetForeignKeyName {
    param (
        [string]$definition
    )

    $type, $name, $modifiers = $definition -split ' ',3
    $foreignKey = ""

    if ($modifiers -match "ForeignKey") {
        $foreignKey += @"
        `n        "${type} ${name}Name"
"@
    }

    return $foreignKey
}

function CreateDirectoryIfNotExists {
    param (
        [string]$path
    )
    if (-not (Test-Path $path)) {
        New-Item -Path $path -ItemType Directory -Force
    }
}

# Base directory path
#".\crop\src\SDDUMP.CropService"


$baseDir = "$folder\${Service}" 
$domainSharedDir = "$baseDir.Domain.Shared"
$domainDir = "$baseDir.Domain"
$contractsDir = "$baseDir.Application.Contracts"
$applicationDir = "$baseDir.Application"
$apiDir = "$baseDir.HttpApi"
$efCoreDir = "$baseDir.EntityFrameworkCore"

# Specific directory paths
$entityDir = "$domainDir\Entities"
$constantsDir = "$domainSharedDir\Constants"
$enumsDir = "$domainSharedDir\Enums"
$eventsDir = "$domainSharedDir\Events"
$permissionsDir = "$contractsDir\Permissions"
$localizationDir = "$domainSharedDir\Localization\${Service}"
$seedingDir = "$domainDir\Data"
$repositoryDir = "$domainDir\Repositories"
$dtoDir = "$contractsDir\$EntityName"
$serviceDir = "$applicationDir\Services"
$mapperDir = "$applicationDir\AutoMapper"
$apiControllerDir = "$apiDir\Controllers"
$efCoreConfigurationsDir = "$efCoreDir\EntityFrameworkCore\Configurations"


# Create all necessary directories
$directories = @(
    $entityDir, $constantsDir, $enumsDir, $eventsDir, $permissionsDir,
    $localizationDir, $seedingDir, $repositoryDir, $dtoDir, $serviceDir, $mapperDir, $apiControllerDir, $efCoreConfigurationsDir
)

foreach ($dir in $directories) {
    CreateDirectoryIfNotExists $dir
}

Write-Host "1. Create Constants"
$constantsFileName = "$constantsDir\${EntityName}Constants.cs"
$constantsContent = @"
namespace ${Service}.Constants
{
    public static class ${EntityName}Constants
    {
        public const string DefaultSorting = "CreationTime desc";
        
        public static class CacheKeys
        {
            public const string ListCacheKey = "All${EntityName}List";
            public const string SingleKey = "${EntityName}";
        }

        public static class ValidationConstants
        {
$(foreach ($attribute in $Attributes) {
    $type, $name, $modifiers = $attribute -split ' ' ,3
    if ($modifiers -match "maxlen:(\d+)") {
        $maxLength = $matches[1]
        "            public const int ${name}MaxLength = ${maxLength};`n"
    }
})
        }
    }
}
"@

New-Item -Path $constantsFileName -ItemType "file" -Force
Set-Content -Path $constantsFileName -Value $constantsContent

Write-Host "2. Create Events"
$eventTypesFileName = "$eventsDir\${EntityName}EtoTypes.cs"
$eventTypesContent = @"
namespace ${Service}.Events
{
    public static class ${EntityName}EtoTypes
    {
        public const string Created = "${Service}.${EntityName}.Created";
        public const string Updated = "${Service}.${EntityName}.Updated";
        public const string Deleted = "${Service}.${EntityName}.Deleted";
    }
}
"@

New-Item -Path $eventTypesFileName -ItemType "file" -Force
Set-Content -Path $eventTypesFileName -Value $eventTypesContent

$etoFileName = "$eventsDir\${EntityName}Eto.cs"
$etoContent = @"
using System;
using Volo.Abp.Domain.Entities.Events.Distributed;

namespace ${Service}.Events
{
    [Serializable]
    public class ${EntityName}Eto : EtoBase
    {
        public long Id { get; set; }
$(foreach ($attribute in $Attributes) {
    $type, $name, $modifiers = $attribute -split ' ',3
    if($modifiers -match "ForeignKey")
    {
        "        public string ${name}Name { get; set; }`n"
    }
    else
    {
        "        public $type $name { get; set; }`n"
    }
})
        public DateTime CreationTime { get; set; }
        public DateTime? LastModificationTime { get; set; }
    }
}
"@

New-Item -Path $etoFileName -ItemType "file" -Force
Set-Content -Path $etoFileName -Value $etoContent


Write-Host "3. Create Permissions"
if ($EntityType -ne "ValueObject") 
{
    # Create Permissions in the application.contracts layer

    $permissionsFileName = "$permissionsDir\${Service}Permissions.cs"
    $existingContent = @"
using Volo.Abp.Reflection;
namespace ${Service}.Permissions
{
    public static class ${Service}Permissions
    {
        public const string GroupName = "${Service}";
        public static string[] GetAll()
        {
            return ReflectionHelper.GetPublicConstantsRecursively(typeof(${Service}Permissions));
        }
    }
}
"@

    # Ensure the file exists
    if (-not (Test-Path $permissionsFileName)) {
        New-Item -Path $permissionsFileName -ItemType File -Force
    } 
    else 
    {
        # Read existing content if file exists
        $existingContent = Get-Content -Path $permissionsFileName -Raw
    }

    # Add new entity permissions before the GetAll() method
    $newPermissions = @"
    public static class ${EntityName}Management
    {
        public const string Default = GroupName + ".${EntityName}";
        public const string Create = Default + ".Create";
        public const string Update = Default + ".Update";
        public const string Delete = Default + ".Delete";
    }

"@
    $existingContent = $existingContent -replace "(\s+public static string\[\] GetAll\(\))", "$newPermissions`$1"
    Set-Content -Path $permissionsFileName -Value $existingContent

    # Update PermissionDefinitionProvider
    $providerFileName = "$permissionsDir\${Service}PermissionDefinitionProvider.cs"

    if (Test-Path $providerFileName) {
        $existingProviderContent = Get-Content -Path $providerFileName -Raw
    } else {
        New-Item -Path $providerFileName -ItemType File -Force
        $existingProviderContent = @"
using Volo.Abp.Authorization.Permissions;
using SDDUMP.${Service}.Localization;
using Volo.Abp.Localization;
namespace ${Service}.Permissions
{
    public class ${Service}PermissionDefinitionProvider : PermissionDefinitionProvider
    {
        private static LocalizableString L(string name)
        {
            return LocalizableString.Create<${Service}Resource>(name);
        }        
        public override void Define(IPermissionDefinitionContext context)
        {
            var ${Service}Group = context.AddGroup(${Service}Permissions.GroupName, L(""Permission:${Service}""));
        }
    }
}
"@

    }
    # Add new entity permissions before the last closing brace
    $newProviderPermissions = @"
        `n
        var ${entityName}Permission = ${Service}Group.AddPermission(
        ${Service}Permissions.${EntityName}Management.Default, L($"Permission:${EntityName}"));
        ${entityName}Permission.AddChild(${Service}Permissions.${EntityName}Management.Create, L($"Permission:${EntityName}.Create"));
        ${entityName}Permission.AddChild(${Service}Permissions.${EntityName}Management.Update,L($"Permission:${EntityName}.Update"));
        ${entityName}Permission.AddChild(${Service}Permissions.${EntityName}Management.Delete, L($"Permission:${EntityName}.Delete"));
"@
    $existingProviderContent = $existingProviderContent -replace "(\s+}\s+}\s+}\s*$)", "$newProviderPermissions`$1"
    Set-Content -Path $providerFileName -Value $existingProviderContent
}


Write-Host "4. Create Entity"
$entityFileName = "$entityDir\$EntityName.cs"
$entityFileContent = @"
using System;
using Volo.Abp.Domain.Entities.Auditing;
using System.ComponentModel.DataAnnotations;
using ${Service}.Events;
using ${Service}.Constants;
using Volo.Abp.Domain.Values;
using System.Collections.Generic;
using Volo.Abp.Domain.Entities;
using System.ComponentModel.DataAnnotations.Schema;

namespace ${Service}.Domain.Entities
{
    public class $EntityName : $EntityType
    {
$(foreach ($attribute in $Attributes) {
    $type, $name, $modifiers = $attribute -split ' ',3
    $annotations = GetAnnotations $attribute
    $foreignKey= GetForeignKey $attribute
    "$annotations$foreignKey
        public $type $name { get; set; }`n"
})    
        protected ${EntityName}() { }

        public ${EntityName}(long id, $(GetOriginalAttributes $Attributes)) $(if($EntityType -ne "ValueObject"){ ": base(id)"})
        {
$(foreach ($attribute in $Attributes) {
    $type, $name, $modifiers = $attribute -split ' ',3
    if ($modifiers -notmatch "ForeignKey") {
        "            Set${name}($(ConvertFirstLetterToLowerCase $name));`n"
    }
})
        }

$(foreach ($attribute in $Attributes) {
    $type, $name, $modifiers = $attribute -split ' ',3
    if($modifiers -notmatch "ForeignKey")
    {
    "        public void Set${name}($type $(ConvertFirstLetterToLowerCase $name)) => $name = $(ConvertFirstLetterToLowerCase $name);
`n"
    }
})
$(if($EntityType -ne "ValueObject" -and $EntityType -ne "Entity<Guid>"){
"
        public void PublishDistributedEvent(${EntityName}Eto eto)
        {
            AddDistributedEvent(eto);
        }
`n"
})
$(if($EntityType -eq "ValueObject"){
"        protected override IEnumerable<object> GetAtomicValues()
        {
$(foreach ($attribute in $Attributes) {
    $type, $name, $modifiers = $attribute -split ' ',3
        "            yield return ${name};`n"
})
        }
`n"
})

        public void Update($(GetOriginalAttributes $Attributes))
        {
$(foreach ($attribute in $Attributes) {
    $type, $name, $modifiers = $attribute -split ' ',3
    if ($modifiers -notmatch "ForeignKey") {
        "            Set${name}($(ConvertFirstLetterToLowerCase $name));`n"
    }
})
        }

    }

}
"@

New-Item -Path $entityFileName -ItemType "file" -Force
Set-Content -Path $entityFileName -Value $entityFileContent

Write-Host "5. Create DTOs"
if($EntityType -ne "ValueObject"){
$createDtoFileName = "$dtoDir\Create${EntityName}Dto.cs"
$createDtoFileContent = @"
using System;
using System.ComponentModel.DataAnnotations;
using ${Service}.Constants;

namespace ${Service}.Application.Contracts.$EntityName
{
    public class Create${EntityName}Dto
    {
$(foreach ($attribute in $Attributes) {
    $annotations = GetAnnotations $attribute
    $type, $name, $modifiers = $attribute -split ' ',3
    if ($modifiers -notmatch "ForeignKey") {
        "$annotations
        public $type $name { get; set; }`n"
    }
})    
    }
}
"@

    New-Item -Path $createDtoFileName -ItemType "file" -Force
    Set-Content -Path $createDtoFileName -Value $createDtoFileContent

$updateDtoFileName = "$dtoDir\Update${EntityName}Dto.cs"
$updateDtoFileContent = @"
using System;
using System.ComponentModel.DataAnnotations;
using ${Service}.Constants;

namespace ${Service}.Application.Contracts.$EntityName
{
    public class Update${EntityName}Dto
    {
$(foreach ($attribute in $Attributes) {
    $annotations = GetAnnotations $attribute
    $type, $name, $modifiers = $attribute -split ' ',3
    if ($modifiers -notmatch "ForeignKey") {
        "$annotations
        public $type $name { get; set; }`n"
    }
})    
    }
}
"@

    New-Item -Path $updateDtoFileName -ItemType "file" -Force
    Set-Content -Path $updateDtoFileName -Value $updateDtoFileContent
}
$readDtoFileName = "$dtoDir\${EntityName}Dto.cs"
$readDtoFileContent = @"
using System;
using Volo.Abp.Application.Dtos;

namespace ${Service}.Application.Contracts.$EntityName
{
    public class ${EntityName}Dto : AuditedEntityDto<long>
    {
$(foreach ($attribute in $Attributes) {
    $type, $name, $modifiers = $attribute -split ' ',3
    if($modifiers -match "ForeignKey")
    {
        "       public string ${name}Name { get; set; }`n"
    }
    else
    {
        "       public $type $name { get; set; }`n"
    }
})    
    }
}
"@

New-Item -Path $readDtoFileName -ItemType "file" -Force
Set-Content -Path $readDtoFileName -Value $readDtoFileContent


Write-Host "6. Create Application Service"
if($EntityType -ne "ValueObject"){
$serviceImplementationFileName = "$serviceDir\${EntityName}AppService.cs"
$serviceImplementationContent = @"
using System;
using Volo.Abp.Application.Services;
using ${Service}.Application.Contracts.Services;
using ${Service}.Domain.Entities;
using ${Service}.Application.Contracts.$EntityName;
using Volo.Abp.Domain.Repositories;
using Microsoft.AspNetCore.Authorization;
using ${Service}.Permissions;
using ${Service}.Constants;
using Volo.Abp.Application.Dtos;
using Volo.Abp.Caching;
using System.Threading.Tasks;
using System.Collections.Generic;
using System.Linq;
using static ${Service}.Permissions.${Service}Permissions;
using ${Service}.Application.Contracts;
using System.Threading;
using ${Service}.Application.Contracts.Shared;
using ${Service}.Events;
using Volo.Abp;

namespace ${Service}.Application.Services
{
    [RemoteService(false)]
    [Authorize(${EntityName}Management.Default)]
    public class ${EntityName}AppService : 
        CrudAppService<
            $EntityName,
            ${EntityName}Dto,
            long,
            SearchedPagedAndSortedResultRequestDto,
            Create${EntityName}Dto,
            Update${EntityName}Dto>,
        I${EntityName}AppService
    {
        private readonly IDistributedCache<${EntityName}Dto> _cache;
        private readonly IDistributedCache<List<${EntityName}Dto>> _listCache;

        public ${EntityName}AppService(
            IRepository<$EntityName, long> repository,
            IDistributedCache<${EntityName}Dto> cache,
            IDistributedCache<List<${EntityName}Dto>> listCache)
            : base(repository)
        {
            _cache = cache;
            _listCache = listCache;

            GetPolicyName = ${EntityName}Management.Default;
            GetListPolicyName = ${EntityName}Management.Default;
            CreatePolicyName = ${EntityName}Management.Create;
            UpdatePolicyName = ${EntityName}Management.Update;
            DeletePolicyName = ${EntityName}Management.Delete;
        }

        public override async Task<${EntityName}Dto> GetAsync(long id)
        {
            var cacheKey = $"{${EntityName}Constants.CacheKeys.SingleKey}:{id}";
            var cachedDto = await _cache.GetAsync(cacheKey);
            
            if (cachedDto != null)
            {
                return cachedDto;
            }

            var entity = await Repository.GetAsync(id);
            var dto = ObjectMapper.Map<${EntityName}, ${EntityName}Dto>(entity);
            
            await _cache.SetAsync(cacheKey, dto);
            
            return dto;
        }

        public override async Task<PagedResultDto<${EntityName}Dto>> GetListAsync(SearchedPagedAndSortedResultRequestDto input)
        {
            if (input.Sorting.IsNullOrWhiteSpace())
            {
                input.Sorting = ${EntityName}Constants.DefaultSorting;
            }

            var result = await base.GetListAsync(input);

            foreach (var dto in result.Items)
            {
                var cacheKey = $"{${EntityName}Constants.CacheKeys.SingleKey}:{dto.Id}";
                await _cache.SetAsync(cacheKey, dto);
            }

            return result;
        }

        public override async Task<${EntityName}Dto> CreateAsync(Create${EntityName}Dto input)
        {
            await CheckCreatePolicyAsync();

            var entity = await MapToEntityAsync(input);

            await Repository.InsertAsync(entity, autoSave: true);

            await _listCache.RemoveAsync(${EntityName}Constants.CacheKeys.ListCacheKey);
$(if($EntityType -ne "ValueObject" -and $EntityType -ne "Entity<Guid>"){
"
            var eto = ObjectMapper.Map<${EntityName}, ${EntityName}Eto>(entity);
            eto.CreationTime = DateTime.UtcNow;
            entity.PublishDistributedEvent(eto);
`n"
})            
            return await MapToGetOutputDtoAsync(entity);
        }

        public override async Task<${EntityName}Dto> UpdateAsync(long id, Update${EntityName}Dto input)
        {
            await CheckUpdatePolicyAsync();

            var entity = await GetEntityByIdAsync(id);

            await MapToEntityAsync(input, entity);
            await Repository.UpdateAsync(entity, autoSave: true);

            await _cache.RemoveAsync($"{${EntityName}Constants.CacheKeys.SingleKey}:{id}");
            await _listCache.RemoveAsync(${EntityName}Constants.CacheKeys.ListCacheKey);

$(if($EntityType -ne "ValueObject" -and $EntityType -ne "Entity<long>"){
"
            var eto = ObjectMapper.Map<${EntityName}, ${EntityName}Eto>(entity);
            eto.LastModificationTime = DateTime.UtcNow;
            entity.PublishDistributedEvent(eto);
`n"
})            
            return await MapToGetOutputDtoAsync(entity);
        }

        public override async Task DeleteAsync(long id)
        {
            await CheckDeletePolicyAsync();

            await Repository.DeleteAsync(id);

            await _cache.RemoveAsync($"{${EntityName}Constants.CacheKeys.SingleKey}:{id}");
            await _listCache.RemoveAsync(${EntityName}Constants.CacheKeys.ListCacheKey);
        }
$(if($EntityType -ne "Entity<long>"){ "
        protected override IQueryable<${EntityName}> ApplyDefaultSorting(IQueryable<${EntityName}> query)
        {
            return query.OrderByDescending(x => x.CreationTime);
        }
"})

        protected override async Task<IQueryable<${EntityName}>> CreateFilteredQueryAsync(SearchedPagedAndSortedResultRequestDto input)
        {
            var data = await base.CreateFilteredQueryAsync(input);
            if (!input.Search.IsNullOrEmpty())
            {
                string searchTerm = input.Search.ToLower();

                data = data
                    .Where(x => x.NameAr.ToLower().Contains(searchTerm)
                             || x.NameEn.ToLower().Contains(searchTerm));
            }
            if (input.ParentId != 0)
            {
                //data = data
                //   .Where(x => x.CountryId == input.ParentId);
            }
            var isArabic = Thread.CurrentThread.CurrentUICulture.Name == "ar";
            if (string.IsNullOrEmpty(input.Sorting))
                data.OrderBy(x => isArabic ? x.NameAr : x.NameEn);
            return data;
        }

    }
}
"@

    New-Item -Path $serviceImplementationFileName -ItemType "file" -Force
    Set-Content -Path $serviceImplementationFileName -Value $serviceImplementationContent

Write-Host "7. Create Repository"
$repositoryFileName = "$repositoryDir\I${EntityName}Repository.cs"
$repositoryContent = @"
using System;
using Volo.Abp.Domain.Repositories;
using ${Service}.Domain.Entities;

namespace ${Service}.Domain.Repositories
{
    public interface I${EntityName}Repository : IRepository<${EntityName}, long>
    {
    }
}
"@

    New-Item -Path $repositoryFileName -ItemType "file" -Force
    Set-Content -Path $repositoryFileName -Value $repositoryContent
}
Write-Host "8. Create AutoMapper Profile"
$mapperProfileFileName = "$mapperDir\${EntityName}Profile.cs"
$mapperProfileContent = @"
using AutoMapper;
using ${Service}.Domain.Entities;
using ${Service}.Application.Contracts.${EntityName};
using ${Service}.Events;

namespace ${Service}.Application.AutoMapper
{
    public class ${EntityName}Profile : Profile
    {
        public ${EntityName}Profile()
        {
            CreateMap<$EntityName, ${EntityName}Dto>().ReverseMap();
$(if($EntityType -ne "ValueObject"){            
"            CreateMap<Create${EntityName}Dto, $EntityName>().ReverseMap();
            CreateMap<Update${EntityName}Dto, $EntityName>().ReverseMap();
"
})            
            CreateMap<${EntityName}, ${EntityName}Eto>().ReverseMap();
        
        }
    }
}
"@

New-Item -Path $mapperProfileFileName -ItemType "file" -Force
Set-Content -Path $mapperProfileFileName -Value $mapperProfileContent

Write-Host "9. Create API Controller"
if($EntityType -ne "ValueObject"){
    $apiControllerFileName = "$apiDir\Controllers\${EntityName}Controller.cs"
    $apiControllerContent = @"
using Microsoft.AspNetCore.Mvc;
using ${Service}.Application.Contracts.Services;
using ${Service}.Application.Contracts.$EntityName;
using System;
using Volo.Abp.AspNetCore.Mvc;
using Volo.Abp.Application.Dtos;
using System.Threading.Tasks;
using Microsoft.AspNetCore.Authorization;
using static ${Service}.Permissions.${Service}Permissions;
using ${Service}.Application.Contracts;
using ${Service}.Application.Contracts.Shared;
using Volo.Abp;

namespace ${Service}.HttpApi.Controllers
{
    [Route("api/$(Get-Plural ${EntityName})")]
    public class ${EntityName}Controller : AbpControllerBase
    {
        private readonly I${EntityName}AppService _appService;

        public ${EntityName}Controller(I${EntityName}AppService appService)
        {
            _appService = appService;
        }

        [HttpGet]
        [Route("{id}")]
        [Authorize(${EntityName}Management.Default)]
        public virtual Task<${EntityName}Dto> GetAsync(long id)
        {
            return _appService.GetAsync(id);
        }

        [HttpGet]
        [Authorize(${EntityName}Management.Default)]
        public virtual Task<PagedResultDto<${EntityName}Dto>> GetListAsync([FromQuery] SearchedPagedAndSortedResultRequestDto input)
        {
            return _appService.GetListAsync(input);
        }

        [HttpPost]
        [Authorize(${EntityName}Management.Create)]
        public virtual Task<${EntityName}Dto> CreateAsync(Create${EntityName}Dto input)
        {
            return _appService.CreateAsync(input);
        }

        [HttpPut]
        [Route("{id}")]
        [Authorize(${EntityName}Management.Update)]
        public virtual Task<${EntityName}Dto> UpdateAsync(long id, Update${EntityName}Dto input)
        {
            return _appService.UpdateAsync(id, input);
        }

        [HttpDelete]
        [Route("{id}")]
        [Authorize(${EntityName}Management.Delete)]
        public virtual Task DeleteAsync(long id)
        {
            return _appService.DeleteAsync(id);
        }
    }
}
"@

    New-Item -Path $apiControllerFileName -ItemType "file" -Force
    Set-Content -Path $apiControllerFileName -Value $apiControllerContent


Write-Host "10. Create Seeder"
    $seederFileName = "$seedingDir\${EntityName}DataSeeder.cs"
    $seederContent = @"
using System;
using System.Threading.Tasks;
using Volo.Abp.Data;
using Volo.Abp.DependencyInjection;
using Volo.Abp.Domain.Repositories;
using ${Service}.Domain.Entities;
using Volo.Abp.Guids;

namespace ${Service}.Domain.Data
{
    public class ${EntityName}DataSeeder : IDataSeedContributor, ITransientDependency
    {
        private readonly IRepository<$EntityName, long> _repository;

        public ${EntityName}DataSeeder(
            IRepository<$EntityName, long> repository)
        {
            _repository = repository;
        }

        public async Task SeedAsync(DataSeedContext context)
        {
            if (await _repository.GetCountAsync() > 0)
            {
                return;
            }

            //await _repository.InsertAsync(new $EntityName(
            //    _guidGenerator.Create()$(foreach ($attribute in $Attributes) {
                    $type, $name, $modifiers = $attribute -split ' ',3
                    if ($modifiers -notmatch "ForeignKey") {
                        switch($type) {
                            "string" { ", `"Sample $name`"" }
                            "int" { ", 0" }
                            "decimal" { ", 0m" }
                            "DateTime" { ", DateTime.Now" }
                            "bool" { ", false" }
                            default { ", default($type)" }
                        }
                    }
                })
           // ), autoSave: true);
        }
    }
}
"@

    New-Item -Path $seederFileName -ItemType "file" -Force
    Set-Content -Path $seederFileName -Value $seederContent

Write-Host "11. Create Application Service Interface"
    $serviceInterfaceFileName = "$contractsDir\Services\I${EntityName}AppService.cs"
    $serviceInterfaceContent = @"
using System;
using Volo.Abp.Application.Services;
using ${Service}.Application.Contracts.$EntityName;
using Volo.Abp.Application.Dtos;
using ${Service}.Application.Contracts.Shared;

namespace ${Service}.Application.Contracts.Services
{
    public interface I${EntityName}AppService : 
        ICrudAppService<
            ${EntityName}Dto,
            long,
            SearchedPagedAndSortedResultRequestDto,
            Create${EntityName}Dto,
            Update${EntityName}Dto>
    {
    }
}
"@

    New-Item -Path $serviceInterfaceFileName -ItemType "file" -Force
    Set-Content -Path $serviceInterfaceFileName -Value $serviceInterfaceContent
}

Write-Host "12. Create entity configuration"

# Ensure Configurations directory exists
if (-not (Test-Path $efCoreConfigurationsDir)) {
    New-Item -Path $efCoreConfigurationsDir -ItemType Directory -Force
}


$configurationFileName = "$efCoreConfigurationsDir\${EntityName}Configuration.cs"
$configurationContent = @"
using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using Volo.Abp.EntityFrameworkCore.Modeling;
using ${Service}.Domain.Entities;


namespace ${Service}.EntityFrameworkCore.Configurations;

public class ${EntityName}Configuration : IEntityTypeConfiguration<${EntityName}>
{
    public void Configure(EntityTypeBuilder<${EntityName}> builder)
    {
        // builder.ToTable(${Service}DbProperties.DbTablePrefix + "$(Get-Plural ${EntityName})", ${Service}DbProperties.DbSchema);

        builder.ConfigureByConvention();

        // builder.Property(x => x.TenantId).IsRequired(false);
        // Configure other properties here
        // Example:
        // builder.Property(x => x.Name).IsRequired().HasMaxLength(128);
    }
}
"@

New-Item -Path $configurationFileName -ItemType "file" -Force
Set-Content -Path $configurationFileName -Value $configurationContent

Write-Host "Add DbSet to DbContext"

$dbContextFile = "$efCoreDir\EntityFrameworkCore\${Service}DbContext.cs"
$dbContextContent = Get-Content -Path $dbContextFile -Raw

# Check if DbSet already exists
if ($dbContextContent -notmatch "DbSet<$EntityName>\s+\w+\s*\{") {

    $dbSetProperty = "        public virtual DbSet<$EntityName> $(Get-Plural $EntityName) { get; set; }`n"

    # Insert after the DbSet marker comment
    $dbContextContent = $dbContextContent -replace `
        "(\/\* Add DbSet properties for your Aggregate Roots \/ Entities here\. \*\/\s*)",
        "`$1$dbSetProperty"

    Set-Content -Path $dbContextFile -Value $dbContextContent -Encoding UTF8
    Write-Host "DbSet<$EntityName> added successfully."
}
else {
    Write-Host "DbSet<$EntityName> already exists. Skipping."
}

Write-Host "Add DbSet to IDbContext"

$IdbContextContent=@"
using Microsoft.EntityFrameworkCore;
using Volo.Abp.Data;
using Volo.Abp.EntityFrameworkCore;
using SDDUMP.ConsultService.Domain.Entities;

namespace ${Service}.EntityFrameworkCore;

[ConnectionStringName(${Service}DbProperties.ConnectionStringName)]
public interface I${Service}DbContext : IEfCoreDbContext
{
    /* Add DbSet for each Aggregate Root here. Example:
     * DbSet<Question> Questions { get; }
     */
    DbSet<${EntityName}> $(Get-Plural ${EntityName}) { get; }
}
"@
$IdbContextFile = "$efCoreDir\EntityFrameworkCore\I${Service}DbContext.cs"
if (-not (Test-Path $IdbContextFile)) {
    New-Item -Path $IdbContextFile -ItemType File -Force
}
else
{
    $IdbContextContent = Get-Content -Path $IdbContextFile -Raw
}
Write-Host "Check if DbSet already exists"
if (-not ($IdbContextContent -match "DbSet<$EntityName>")) {

    # Add DbSet property
    $IdbSetProperty = "`n    DbSet<$EntityName> $(Get-Plural ${EntityName}) { get; }`n"
    $IdbContextContent = $IdbContextContent -replace "(\s+public I${Service}DbContext\()", "$IdbSetProperty`n`$1"
    
    Set-Content -Path $dbContextFile -Value $dbContextContent
}

Write-Host "Update OnModelCreating"

$modelCreatingPattern = "protected override void OnModelCreating\(ModelBuilder builder\)\s*\{"

if ($dbContextContent -match $modelCreatingPattern) {

    # Add configuration if not already present
    if (-not ($dbContextContent -match "builder\.ApplyConfiguration\(new ${EntityName}Configuration\(\)\);")) {

        $newConfigLine = "            builder.ApplyConfiguration(new ${EntityName}Configuration());`n"

        # Insert after method signature line
        $dbContextContent = $dbContextContent -replace "($modelCreatingPattern)", "$1`n$newConfigLine"

        # Add using statement for Configurations namespace if missing
        $usingStatement = "using ${Service}.EntityFrameworkCore.Configurations;"
        if (-not ($dbContextContent -match [regex]::Escape($usingStatement))) {
            # Insert after the last using statement
            $dbContextContent = $dbContextContent -replace "((using .+;\s*)+)(?=\s*namespace)", "$1$usingStatement`n"
        }

        # Save changes
        Set-Content -Path $dbContextFile -Value $dbContextContent -Encoding UTF8
        Write-Host "Configuration for $EntityName added successfully."
    }
    else {
        Write-Host "Configuration for $EntityName already exists. Skipping."
    }
}

Write-Host "Optional: Create/Update Repository if custom repository is needed"

    $repositoryDir = "$efCoreDir\EntityFrameworkCore\Repositories"
    if (-not (Test-Path $repositoryDir)) {
        New-Item -Path $repositoryDir -ItemType Directory -Force
    }

    $repositoryFileName = "$repositoryDir\EfCore${EntityName}Repository.cs"
    $repositoryContent = @"
using ${Service}.Domain.Entities;
using ${Service}.Domain.Repositories;
using System;
using System.Linq;
using System.Threading.Tasks;
using Volo.Abp.Domain.Repositories.EntityFrameworkCore;
using Volo.Abp.EntityFrameworkCore;

namespace ${Service}.EntityFrameworkCore.Repositories;

public class EfCore${EntityName}Repository : EfCoreRepository<${Service}DbContext, ${EntityName}, long>, I${EntityName}Repository
{
    public EfCore${EntityName}Repository(IDbContextProvider<${Service}DbContext> dbContextProvider)
        : base(dbContextProvider)
    {
    }

    // Add custom repository methods here
}
"@

    New-Item -Path $repositoryFileName -ItemType "file" -Force
    Set-Content -Path $repositoryFileName -Value $repositoryContent


Write-Host "13. Update Localization "
# Function to safely add translation if it doesn't exist
function Add-TranslationIfMissing {
    param (
        [string]$key,
        [string]$value
    )
    if (-not $localization.texts.ContainsKey($key)) {
        $localization.texts[$key] = $value
    }
}

#en
$localizationFile = "$localizationDir\en.json"
if (Test-Path $localizationFile) {
    $localization = Get-Content -Path $localizationFile -Raw | ConvertFrom-Json
    
    # Convert the PSObject to a hashtable
    if ($localization.texts -isnot [System.Collections.Hashtable]) {
        $textsHashtable = @{}
        if ($localization.texts) {
            $localization.texts.PSObject.Properties | ForEach-Object {
                $textsHashtable[$_.Name] = $_.Value
            }
        }
        $localization.texts = $textsHashtable
    }
} else {
    $localization = [PSCustomObject]@{
        culture = "en"
        texts = @{}
    }
}

Write-Host "Add the translations only if they don't exist"
Add-TranslationIfMissing "$EntityName" $EntityName
Add-TranslationIfMissing "${EntityName}Management" "$EntityName Management"
Add-TranslationIfMissing "Create$EntityName" "Create $EntityName"
Add-TranslationIfMissing "Edit$EntityName" "Edit $EntityName"
Add-TranslationIfMissing "Delete$EntityName" "Delete $EntityName"
Add-TranslationIfMissing "${EntityName}DeletionConfirmationMessage" "Are you sure to delete this $EntityName?"

foreach ($attribute in $Attributes) {
    $type, $name, $modifiers = $attribute -split ' ',3
    Add-TranslationIfMissing "$EntityName.$name" $name
}

Write-Host "Create ordered version of the texts"
$orderedTexts = [ordered]@{}
$localization.texts.GetEnumerator() | Sort-Object -Property Key | ForEach-Object {
    $orderedTexts[$_.Key] = $_.Value
}

# Replace texts with ordered version
$localization.texts = $orderedTexts

# Save changes with ordered texts
$localization | ConvertTo-Json -Depth 10 | Set-Content -Path $localizationFile

#ar
$localizationFile = "$localizationDir\ar.json"
if (Test-Path $localizationFile) {
    $localization = Get-Content -Path $localizationFile -Raw | ConvertFrom-Json
    
    # Convert the PSObject to a hashtable
    if ($localization.texts -isnot [System.Collections.Hashtable]) {
        $textsHashtable = @{}
        if ($localization.texts) {
            $localization.texts.PSObject.Properties | ForEach-Object {
                $textsHashtable[$_.Name] = $_.Value
            }
        }
        $localization.texts = $textsHashtable
    }
} else {
    $localization = [PSCustomObject]@{
        culture = "en"
        texts = @{}
    }
}

# Add the translations only if they don't exist
Add-TranslationIfMissing "$EntityName" $EntityName
Add-TranslationIfMissing "${EntityName}Management" "$EntityName Management"
Add-TranslationIfMissing "Create$EntityName" "Create $EntityName"
Add-TranslationIfMissing "Edit$EntityName" "Edit $EntityName"
Add-TranslationIfMissing "Delete$EntityName" "Delete $EntityName"
Add-TranslationIfMissing "${EntityName}DeletionConfirmationMessage" "Are you sure to delete this $EntityName?"

foreach ($attribute in $Attributes) {
    $type, $name, $modifiers = $attribute -split ' ',3
    Add-TranslationIfMissing "$EntityName.$name" $name
}

# Create ordered version of the texts
$orderedTexts = [ordered]@{}
$localization.texts.GetEnumerator() | Sort-Object -Property Key | ForEach-Object {
    $orderedTexts[$_.Key] = $_.Value
}

# Replace texts with ordered version
$localization.texts = $orderedTexts

# Save changes with ordered texts
$localization | ConvertTo-Json -Depth 10 | Set-Content -Path $localizationFile


Write-Host "Entity $EntityName has been created successfully with all necessary components."
Write-Host "Don't forget to:"
Write-Host "1. Register the new permissions in PermissionDefinitionProvider"
Write-Host "2. Add the DbSet to your DbContext"
Write-Host "3. Create the necessary database migration"
Write-Host "4. Update your menu configuration if needed"
