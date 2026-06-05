package abpcrud

import (
	"fmt"
	"strings"
)

const (
	templateConstants                    = "constants.cs.tmpl"
	templateEventTypes                   = "event-types.cs.tmpl"
	templateEto                          = "eto.cs.tmpl"
	templateEntity                       = "entity.cs.tmpl"
	templateCreateDto                    = "create-dto.cs.tmpl"
	templateUpdateDto                    = "update-dto.cs.tmpl"
	templateEntityDto                    = "entity-dto.cs.tmpl"
	templateRepositoryInterface          = "repository-interface.cs.tmpl"
	templateAppServiceInterface          = "appservice-interface.cs.tmpl"
	templatePermissions                  = "permissions.cs.tmpl"
	templatePermissionDefinitionProvider = "permission-definition-provider.cs.tmpl"
	templateAppService                   = "appservice.cs.tmpl"
	templateAutoMapperProfile            = "automapper-profile.cs.tmpl"
	templateMapperlyMappers              = "mapperly-mappers.cs.tmpl"
	templateController                   = "controller.cs.tmpl"
	templateDataSeeder                   = "data-seeder.cs.tmpl"
	templateEFConfiguration              = "ef-configuration.cs.tmpl"
	templateEFRepository                 = "ef-repository.cs.tmpl"
	templateIDbContext                   = "idbcontext.cs.tmpl"
	templateLocalization                 = "localization.json.tmpl"
)

func constantsTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateConstants, info, opts, templateOptions{})
}

func eventTypesTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateEventTypes, info, opts, templateOptions{})
}

func etoTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateEto, info, opts, templateOptions{})
}

func entityTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateEntity, info, opts, templateOptions{})
}

func createDtoTemplate(info AppInfo, opts Options) (string, error) {
	return dtoInputTemplate(info, opts, "Create"+opts.Entity+"Dto", templateCreateDto)
}

func updateDtoTemplate(info AppInfo, opts Options) (string, error) {
	return dtoInputTemplate(info, opts, "Update"+opts.Entity+"Dto", templateUpdateDto)
}

func dtoInputTemplate(info AppInfo, opts Options, name string, templateName string) (string, error) {
	return renderABPTemplate(templateName, info, opts, templateOptions{DtoName: name})
}

func readDtoTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateEntityDto, info, opts, templateOptions{})
}

func repositoryInterfaceTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateRepositoryInterface, info, opts, templateOptions{})
}

func appServiceInterfaceTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateAppServiceInterface, info, opts, templateOptions{})
}

func permissionsTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templatePermissions, info, opts, templateOptions{})
}

func permissionDefinitionProviderTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templatePermissionDefinitionProvider, info, opts, templateOptions{})
}

func appServiceTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateAppService, info, opts, templateOptions{})
}

func autoMapperProfileTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateAutoMapperProfile, info, opts, templateOptions{})
}

func mapperlyMappersTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateMapperlyMappers, info, opts, templateOptions{})
}

func controllerTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateController, info, opts, templateOptions{})
}

func dataSeederTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateDataSeeder, info, opts, templateOptions{})
}

func efConfigurationTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateEFConfiguration, info, opts, templateOptions{})
}

func efRepositoryTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateEFRepository, info, opts, templateOptions{})
}

func iDbContextTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateIDbContext, info, opts, templateOptions{})
}

func localizationTemplate(info AppInfo, opts Options, culture string) (string, error) {
	return renderABPTemplate(templateLocalization, info, opts, templateOptions{Culture: culture})
}

func constantsValidationBlock(opts Options) string {
	var validations strings.Builder
	for _, attr := range opts.Attrs {
		if attr.MaxLength > 0 {
			fmt.Fprintf(&validations, "            public const int %sMaxLength = %d;\n", attr.Name, attr.MaxLength)
		}
	}
	return validations.String()
}

func etoPropertiesBlock(opts Options) string {
	var props strings.Builder
	fmt.Fprintf(&props, "        public %s Id { get; set; }\n", opts.KeyType)
	writeAttributes(&props, opts.Attrs, "        ", true)
	if entityHasCreationAudit(opts.EntityType) {
		fmt.Fprintln(&props, "        public DateTime CreationTime { get; set; }")
	}
	if entityHasModificationAudit(opts.EntityType) {
		fmt.Fprintln(&props, "        public DateTime? LastModificationTime { get; set; }")
	}
	return props.String()
}

func entityPropertiesBlock(opts Options) string {
	var props strings.Builder
	for _, attr := range opts.Attrs {
		writeAnnotations(&props, attr, opts.Entity, "        ")
		fmt.Fprintf(&props, "        public %s %s { get; private set; }%s\n\n", attr.Type, attr.Name, csharpPropertyDefault(attr.Type))
	}
	return props.String()
}

func inputPropertiesBlock(opts Options) string {
	var props strings.Builder
	for _, attr := range opts.Attrs {
		writeAnnotations(&props, attr, opts.Entity, "        ")
		fmt.Fprintf(&props, "        public %s %s { get; set; }%s\n\n", attr.Type, attr.Name, csharpPropertyDefault(attr.Type))
	}
	return props.String()
}

func readDtoPropertiesBlock(opts Options) string {
	var props strings.Builder
	writeAttributes(&props, opts.Attrs, "        ", true)
	return props.String()
}

func constructorArgs(opts Options) string {
	if len(opts.Attrs) == 0 {
		return ""
	}
	var args []string
	for _, attr := range opts.Attrs {
		args = append(args, attr.Type+" "+lowerFirst(attr.Name))
	}
	return ", " + strings.Join(args, ", ")
}

func constructorSetsBlock(opts Options) string {
	var constructorSets strings.Builder
	for _, attr := range opts.Attrs {
		fmt.Fprintf(&constructorSets, "            Set%s(%s);\n", attr.Name, lowerFirst(attr.Name))
	}
	return constructorSets.String()
}

func entitySettersBlock(opts Options) string {
	var setters strings.Builder
	for _, attr := range opts.Attrs {
		fmt.Fprintf(&setters, "        public void Set%s(%s %s)\n", attr.Name, attr.Type, lowerFirst(attr.Name))
		fmt.Fprintln(&setters, "        {")
		fmt.Fprintf(&setters, "            %s = %s;\n", attr.Name, lowerFirst(attr.Name))
		fmt.Fprintln(&setters, "        }")
		fmt.Fprintln(&setters)
	}
	return setters.String()
}

func updateArgs(opts Options) string {
	var args []string
	for _, attr := range opts.Attrs {
		args = append(args, attr.Type+" "+lowerFirst(attr.Name))
	}
	return strings.Join(args, ", ")
}

func appServiceCreateArgs(opts Options) string {
	args := []string{newEntityKeyExpression(opts)}
	for _, attr := range opts.Attrs {
		args = append(args, "input."+attr.Name)
	}
	return strings.Join(args, ", ")
}

func appServiceUpdateArgs(opts Options) string {
	args := make([]string, 0, len(opts.Attrs))
	for _, attr := range opts.Attrs {
		args = append(args, "input."+attr.Name)
	}
	return strings.Join(args, ", ")
}

func efPropertyConfigBlock(opts Options) string {
	var propertyConfig strings.Builder
	for _, attr := range opts.Attrs {
		if attr.Required || attr.MaxLength > 0 {
			fmt.Fprintf(&propertyConfig, "            builder.Property(x => x.%s)", attr.Name)
			if attr.Required {
				fmt.Fprint(&propertyConfig, ".IsRequired()")
			}
			if attr.MaxLength > 0 {
				fmt.Fprintf(&propertyConfig, ".HasMaxLength(%sConstants.ValidationConstants.%sMaxLength)", opts.Entity, attr.Name)
			}
			fmt.Fprintln(&propertyConfig, ";")
		}
	}
	return propertyConfig.String()
}

func attributeUsingBlock(attrs []Attribute) string {
	namespaces := map[string]bool{}
	for _, attr := range attrs {
		if usesGenericCollections(attr.Type) {
			namespaces["System.Collections.Generic"] = true
		}
	}
	if len(namespaces) == 0 {
		return ""
	}

	ordered := []string{"System.Collections.Generic"}
	var builder strings.Builder
	for _, namespace := range ordered {
		if namespaces[namespace] {
			fmt.Fprintf(&builder, "using %s;\n", namespace)
		}
	}
	return builder.String()
}

func usesGenericCollections(typ string) bool {
	normalized := strings.ToLower(strings.TrimSpace(typ))
	for _, marker := range []string{
		"list<",
		"ilist<",
		"icollection<",
		"ireadonlycollection<",
		"ireadonlylist<",
		"ienumerable<",
		"dictionary<",
		"idictionary<",
		"hashset<",
		"collection<",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func newEntityKeyExpression(opts Options) string {
	switch strings.ToLower(strings.TrimSpace(opts.KeyType)) {
	case "guid", "system.guid":
		return "GuidGenerator.Create()"
	case "string":
		return `GuidGenerator.Create().ToString("N")`
	default:
		return "default"
	}
}

func dtoBaseClass(entityType string) string {
	normalized := strings.ToLower(entityType)
	switch {
	case strings.Contains(normalized, "fullaudited"):
		return "FullAuditedEntityDto"
	case strings.Contains(normalized, "creationaudited"):
		return "CreationAuditedEntityDto"
	case strings.Contains(normalized, "audited"):
		return "AuditedEntityDto"
	default:
		return "EntityDto"
	}
}

func entityHasCreationAudit(entityType string) bool {
	normalized := strings.ToLower(entityType)
	return strings.Contains(normalized, "audited") || strings.Contains(normalized, "creationaudited")
}

func entityHasModificationAudit(entityType string) bool {
	normalized := strings.ToLower(entityType)
	return strings.Contains(normalized, "fullaudited") ||
		(strings.Contains(normalized, "audited") && !strings.Contains(normalized, "creationaudited"))
}

func writeAttributes(builder *strings.Builder, attrs []Attribute, indent string, includeDefaults bool) {
	for _, attr := range attrs {
		fmt.Fprintf(builder, "%spublic %s %s { get; set; }", indent, attr.Type, attr.Name)
		if includeDefaults {
			builder.WriteString(csharpPropertyDefault(attr.Type))
		}
		builder.WriteString("\n")
	}
}

func writeAnnotations(builder *strings.Builder, attr Attribute, entity string, indent string) {
	if attr.Required {
		fmt.Fprintf(builder, "%s[Required]\n", indent)
	}
	if attr.MaxLength > 0 {
		fmt.Fprintf(builder, "%s[MaxLength(%sConstants.ValidationConstants.%sMaxLength)]\n", indent, entity, attr.Name)
	}
}

func csharpPropertyDefault(typ string) string {
	if strings.EqualFold(typ, "string") {
		return " = string.Empty;"
	}
	return ""
}
