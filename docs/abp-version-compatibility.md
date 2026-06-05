# ABP Version Compatibility

This generator supports ABP 8, 9, and 10 by detecting the installed ABP package version and object-mapping integration.

## Generation Decisions

| ABP version | Runtime generation impact | Generator behavior |
| --- | --- | --- |
| ABP 8 | ABP targets .NET 8. Read-only repository interfaces were introduced, but CRUD application services still use writable `IRepository<TEntity, TKey>`. | Generate standard CRUD files and AutoMapper profiles by default. |
| ABP 9 | ABP targets .NET 9. Static asset host changes are upgrade concerns, not CRUD file changes. | Generate standard CRUD files and AutoMapper profiles by default unless Mapperly markers are found. |
| ABP 10 | ABP targets .NET 10. ABP modules moved from AutoMapper to Mapperly; ABP still supports both integrations. | Generate Mapperly mapper classes by default unless AutoMapper markers are found. |

## Mapping Detection

The CLI scans project files for mapping markers:

```text
Volo.Abp.AutoMapper
AbpAutoMapperModule
AddAutoMapperObjectMapper
AutoMapper Profile classes
Volo.Abp.Mapperly
AbpMapperlyModule
AddMapperlyObjectMapper
Riok.Mapperly
MapperBase / TwoWayMapperBase classes
```

Use `--mapping` when a mixed project needs an explicit choice:

```bash
abp-cli generate --root /path/to/app --entity Product --mapping mapperly --dry-run
abp-cli generate --root /path/to/app --entity Product --mapping automapper --dry-run
```

## Sources

- ABP 10 migration guide: https://abp.io/docs/latest/release-info/migration-guides/abp-10-0
- ABP 9 migration guide: https://abp.io/docs/10.4/release-info/migration-guides/abp-9-0
- ABP 8 migration guide: https://abp.io/docs/8.0/Migration-Guides/Abp-8_0
- ABP AutoMapper to Mapperly migration guide: https://abp.io/docs/latest/release-info/migration-guides/AutoMapper-To-Mapperly
- ABP object mapping docs: https://abp.io/docs/en/abp/latest/Object-To-Object-Mapping
