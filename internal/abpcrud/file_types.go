package abpcrud

import (
	"fmt"
	"strings"
)

const (
	fileTypeConstants                    = "constants"
	fileTypeEventTypes                   = "event-types"
	fileTypeEto                          = "eto"
	fileTypeEntity                       = "entity"
	fileTypeRepositoryInterface          = "repository-interface"
	fileTypeDataSeeder                   = "data-seeder"
	fileTypeCreateDto                    = "create-dto"
	fileTypeUpdateDto                    = "update-dto"
	fileTypeEntityDto                    = "entity-dto"
	fileTypeAppServiceInterface          = "appservice-interface"
	fileTypePermissions                  = "permissions"
	fileTypePermissionDefinitionProvider = "permission-definition-provider"
	fileTypeAppService                   = "appservice"
	fileTypeMapping                      = "mapping"
	fileTypeController                   = "controller"
	fileTypeEFConfiguration              = "ef-configuration"
	fileTypeEFRepository                 = "ef-repository"
	fileTypeDbContext                    = "dbcontext"
	fileTypeIDbContext                   = "idbcontext"
	fileTypeLocalizationEN               = "localization-en"
	fileTypeLocalizationAR               = "localization-ar"
)

var allFileTypes = []string{
	fileTypeConstants,
	fileTypeEventTypes,
	fileTypeEto,
	fileTypeEntity,
	fileTypeRepositoryInterface,
	fileTypeDataSeeder,
	fileTypeCreateDto,
	fileTypeUpdateDto,
	fileTypeEntityDto,
	fileTypeAppServiceInterface,
	fileTypePermissions,
	fileTypePermissionDefinitionProvider,
	fileTypeAppService,
	fileTypeMapping,
	fileTypeController,
	fileTypeEFConfiguration,
	fileTypeEFRepository,
	fileTypeDbContext,
	fileTypeIDbContext,
	fileTypeLocalizationEN,
	fileTypeLocalizationAR,
}

var fileTypeAliases = map[string][]string{
	"all": allFileTypes,

	"domain-shared": {
		fileTypeConstants,
		fileTypeEventTypes,
		fileTypeEto,
		fileTypeLocalizationEN,
		fileTypeLocalizationAR,
	},
	"domainshared": {
		fileTypeConstants,
		fileTypeEventTypes,
		fileTypeEto,
		fileTypeLocalizationEN,
		fileTypeLocalizationAR,
	},
	"domain": {
		fileTypeEntity,
		fileTypeRepositoryInterface,
		fileTypeDataSeeder,
	},
	"application-contracts": {
		fileTypeCreateDto,
		fileTypeUpdateDto,
		fileTypeEntityDto,
		fileTypeAppServiceInterface,
		fileTypePermissions,
		fileTypePermissionDefinitionProvider,
	},
	"contracts": {
		fileTypeCreateDto,
		fileTypeUpdateDto,
		fileTypeEntityDto,
		fileTypeAppServiceInterface,
		fileTypePermissions,
		fileTypePermissionDefinitionProvider,
	},
	"application": {
		fileTypeAppService,
		fileTypeMapping,
	},
	"httpapi": {
		fileTypeController,
	},
	"http-api": {
		fileTypeController,
	},
	"api": {
		fileTypeController,
	},
	"efcore": {
		fileTypeEFConfiguration,
		fileTypeEFRepository,
		fileTypeDbContext,
		fileTypeIDbContext,
	},
	"entityframeworkcore": {
		fileTypeEFConfiguration,
		fileTypeEFRepository,
		fileTypeDbContext,
		fileTypeIDbContext,
	},
	"entity-framework-core": {
		fileTypeEFConfiguration,
		fileTypeEFRepository,
		fileTypeDbContext,
		fileTypeIDbContext,
	},

	"constants":                      {fileTypeConstants},
	"constant":                       {fileTypeConstants},
	"events":                         {fileTypeEventTypes, fileTypeEto},
	"event":                          {fileTypeEventTypes, fileTypeEto},
	"event-types":                    {fileTypeEventTypes},
	"eto-types":                      {fileTypeEventTypes},
	"event-type":                     {fileTypeEventTypes},
	"eto":                            {fileTypeEto},
	"event-dto":                      {fileTypeEto},
	"entity":                         {fileTypeEntity},
	"entities":                       {fileTypeEntity},
	"repository-interface":           {fileTypeRepositoryInterface},
	"domain-repository":              {fileTypeRepositoryInterface},
	"irepository":                    {fileTypeRepositoryInterface},
	"repository":                     {fileTypeRepositoryInterface, fileTypeEFRepository},
	"repositories":                   {fileTypeRepositoryInterface, fileTypeEFRepository},
	"data-seeder":                    {fileTypeDataSeeder},
	"seeder":                         {fileTypeDataSeeder},
	"seed":                           {fileTypeDataSeeder},
	"create-dto":                     {fileTypeCreateDto},
	"update-dto":                     {fileTypeUpdateDto},
	"entity-dto":                     {fileTypeEntityDto},
	"read-dto":                       {fileTypeEntityDto},
	"dto":                            {fileTypeCreateDto, fileTypeUpdateDto, fileTypeEntityDto},
	"dtos":                           {fileTypeCreateDto, fileTypeUpdateDto, fileTypeEntityDto},
	"appservice-interface":           {fileTypeAppServiceInterface},
	"app-service-interface":          {fileTypeAppServiceInterface},
	"service-interface":              {fileTypeAppServiceInterface},
	"permissions":                    {fileTypePermissions, fileTypePermissionDefinitionProvider},
	"permission":                     {fileTypePermissions, fileTypePermissionDefinitionProvider},
	"permissions-class":              {fileTypePermissions},
	"permission-class":               {fileTypePermissions},
	"permission-definition-provider": {fileTypePermissionDefinitionProvider},
	"permission-provider":            {fileTypePermissionDefinitionProvider},
	"appservice":                     {fileTypeAppService},
	"app-service":                    {fileTypeAppService},
	"service":                        {fileTypeAppService},
	"mapping":                        {fileTypeMapping},
	"mapper":                         {fileTypeMapping},
	"mappers":                        {fileTypeMapping},
	"automapper":                     {fileTypeMapping},
	"mapperly":                       {fileTypeMapping},
	"controller":                     {fileTypeController},
	"controllers":                    {fileTypeController},
	"ef-configuration":               {fileTypeEFConfiguration},
	"entity-configuration":           {fileTypeEFConfiguration},
	"configuration":                  {fileTypeEFConfiguration},
	"ef-repository":                  {fileTypeEFRepository},
	"efcore-repository":              {fileTypeEFRepository},
	"dbcontext":                      {fileTypeDbContext},
	"db-context":                     {fileTypeDbContext},
	"idbcontext":                     {fileTypeIDbContext},
	"i-dbcontext":                    {fileTypeIDbContext},
	"dbcontexts":                     {fileTypeDbContext, fileTypeIDbContext},
	"localization":                   {fileTypeLocalizationEN, fileTypeLocalizationAR},
	"localisation":                   {fileTypeLocalizationEN, fileTypeLocalizationAR},
	"localization-en":                {fileTypeLocalizationEN},
	"localisation-en":                {fileTypeLocalizationEN},
	"localization-ar":                {fileTypeLocalizationAR},
	"localisation-ar":                {fileTypeLocalizationAR},
}

func normalizeFileTypes(values []string) ([]string, error) {
	if len(values) == 0 {
		return append([]string(nil), allFileTypes...), nil
	}

	selected := map[string]bool{}
	for _, value := range values {
		for _, token := range splitFileTypeTokens(value) {
			key := normalizeFileTypeToken(token)
			if key == "" {
				continue
			}
			expanded, ok := fileTypeAliases[key]
			if !ok {
				return nil, fmt.Errorf("unsupported file type %q. Supported values include: %s", token, supportedFileTypesText())
			}
			for _, fileType := range expanded {
				selected[fileType] = true
			}
		}
	}

	if len(selected) == 0 {
		return append([]string(nil), allFileTypes...), nil
	}

	normalized := make([]string, 0, len(selected))
	for _, fileType := range allFileTypes {
		if selected[fileType] {
			normalized = append(normalized, fileType)
		}
	}
	return normalized, nil
}

func splitFileTypeTokens(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';'
	})
}

func normalizeFileTypeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	value = strings.NewReplacer("_", "-", ".", "-", "/", "-", "\\", "-", " ", "-").Replace(value)
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

func supportedFileTypesText() string {
	return strings.Join([]string{
		"all",
		"domain-shared",
		"domain",
		"contracts",
		"application",
		"httpapi",
		"efcore",
		"localization",
		"entity",
		"dto",
		"permissions",
		"appservice",
		"mapping",
		"controller",
		"dbcontext",
	}, ", ")
}

func includesFileType(opts Options, fileType string) bool {
	for _, selected := range opts.Files {
		if selected == fileType {
			return true
		}
	}
	return false
}
