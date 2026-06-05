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
	templateGetListInput                 = "get-list-input.cs.tmpl"
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
	templateManyToManyJoinEntity         = "many-to-many-join-entity.cs.tmpl"
	templateManyToManyJoinConfiguration  = "many-to-many-join-configuration.cs.tmpl"
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

// getListInputTemplate renders the generated request DTO used by filtered list
// endpoints.
func getListInputTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateGetListInput, info, opts, templateOptions{})
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

// manyToManyJoinEntityTemplate renders the explicit join entity requested by a
// many-to-many relation config.
func manyToManyJoinEntityTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateManyToManyJoinEntity, info, opts, templateOptions{})
}

// manyToManyJoinConfigurationTemplate renders the EF Core configuration for an
// explicitly named many-to-many join entity.
func manyToManyJoinConfigurationTemplate(info AppInfo, opts Options) (string, error) {
	return renderABPTemplate(templateManyToManyJoinConfiguration, info, opts, templateOptions{})
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
		if attr.Relation.Kind == relationCollection || attr.Relation.Kind == relationManyToMany {
			continue
		}
		if attr.MaxLength > 0 {
			fmt.Fprintf(&validations, "            public const int %sMaxLength = %d;\n", attr.Name, attr.MaxLength)
		}
	}
	return validations.String()
}

func etoPropertiesBlock(opts Options) string {
	var props strings.Builder
	fmt.Fprintf(&props, "        public %s Id { get; set; }\n", opts.KeyType)
	writeAttributes(&props, dtoAttributes(opts.Attrs), "        ", true)
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
		if isToManyRelation(attr) {
			continue
		}
		writeAnnotations(&props, attr, opts.Entity, "        ")
		fmt.Fprintf(&props, "        public %s %s { get; private set; }%s\n\n", attr.Type, attr.Name, csharpPropertyDefault(attr.Type))
		if attr.Relation.Kind == relationReference {
			fmt.Fprintf(&props, "        public %s? %s { get; private set; }\n\n", attr.Relation.Entity, attr.Relation.Navigation)
		}
	}
	for _, attr := range opts.Attrs {
		if isToManyRelation(attr) {
			fmt.Fprintf(
				&props,
				"        public ICollection<%s> %s { get; private set; } = new List<%s>();\n\n",
				attr.Relation.Entity,
				attr.Name,
				attr.Relation.Entity,
			)
		}
	}
	return props.String()
}

func inputPropertiesBlock(opts Options) string {
	var props strings.Builder
	for _, attr := range opts.Attrs {
		if isToManyRelation(attr) {
			continue
		}
		writeAnnotations(&props, attr, opts.Entity, "        ")
		fmt.Fprintf(&props, "        public %s %s { get; set; }%s\n\n", attr.Type, attr.Name, csharpPropertyDefault(attr.Type))
	}
	return props.String()
}

func readDtoPropertiesBlock(opts Options) string {
	var props strings.Builder
	writeAttributes(&props, dtoAttributes(opts.Attrs), "        ", true)
	return props.String()
}

// getListInputName centralizes the list input type name used across contracts,
// application services, and controllers.
func getListInputName(opts Options) string {
	return "Get" + opts.Entity + "ListInput"
}

// getListInputPropertiesBlock emits the generated list filter DTO fields. The
// broad Filter property searches string fields; per-field filters are exact
// matches for simple scalar types.
func getListInputPropertiesBlock(opts Options) string {
	var props strings.Builder
	fmt.Fprintln(&props, "        public string? Filter { get; set; }")
	fmt.Fprintln(&props)
	for _, attr := range opts.Attrs {
		if !isFilterableAttribute(attr) {
			continue
		}
		fmt.Fprintf(
			&props,
			"        public %s? %s { get; set; }\n\n",
			nullableFilterType(attr.Type),
			attr.Name,
		)
	}
	return props.String()
}

// filterPredicatesBlock renders ABP WhereIf predicates for generated list
// queries while keeping collection relations out of scalar filters.
func filterPredicatesBlock(opts Options) string {
	var predicates strings.Builder
	var stringFilters []Attribute
	for _, attr := range opts.Attrs {
		if !isFilterableAttribute(attr) {
			continue
		}
		if isCSharpString(attr.Type) {
			stringFilters = append(stringFilters, attr)
		}
	}
	if len(stringFilters) > 0 {
		fmt.Fprintln(&predicates, "                .WhereIf(!input.Filter.IsNullOrWhiteSpace(), x =>")
		for i, attr := range stringFilters {
			suffix := " ||"
			if i == len(stringFilters)-1 {
				suffix = ")"
			}
			fmt.Fprintf(&predicates, "                    x.%s.Contains(input.Filter!)%s\n", attr.Name, suffix)
		}
	}
	for _, attr := range opts.Attrs {
		if !isFilterableAttribute(attr) {
			continue
		}
		if isCSharpString(attr.Type) {
			fmt.Fprintf(
				&predicates,
				"                .WhereIf(!input.%s.IsNullOrWhiteSpace(), x => x.%s.Contains(input.%s!))\n",
				attr.Name,
				attr.Name,
				attr.Name,
			)
			continue
		}
		fmt.Fprintf(
			&predicates,
			"                .WhereIf(input.%s.HasValue, x => x.%s == input.%s!.Value)\n",
			attr.Name,
			attr.Name,
			attr.Name,
		)
	}
	return predicates.String()
}

func constructorArgs(opts Options) string {
	if len(opts.Attrs) == 0 {
		return ""
	}
	var args []string
	for _, attr := range opts.Attrs {
		if isToManyRelation(attr) {
			continue
		}
		args = append(args, attr.Type+" "+lowerFirst(attr.Name))
	}
	return ", " + strings.Join(args, ", ")
}

func constructorSetsBlock(opts Options) string {
	var constructorSets strings.Builder
	for _, attr := range opts.Attrs {
		if isToManyRelation(attr) {
			continue
		}
		fmt.Fprintf(&constructorSets, "            Set%s(%s);\n", attr.Name, lowerFirst(attr.Name))
	}
	return constructorSets.String()
}

func entitySettersBlock(opts Options) string {
	var setters strings.Builder
	for _, attr := range opts.Attrs {
		if isToManyRelation(attr) {
			continue
		}
		fmt.Fprintf(&setters, "        public void Set%s(%s %s)\n", attr.Name, attr.Type, lowerFirst(attr.Name))
		fmt.Fprintln(&setters, "        {")
		fmt.Fprintf(&setters, "            %s = %s;\n", attr.Name, assignmentExpression(attr))
		fmt.Fprintln(&setters, "        }")
		fmt.Fprintln(&setters)
	}
	return setters.String()
}

func updateArgs(opts Options) string {
	var args []string
	for _, attr := range opts.Attrs {
		if isToManyRelation(attr) {
			continue
		}
		args = append(args, attr.Type+" "+lowerFirst(attr.Name))
	}
	return strings.Join(args, ", ")
}

func appServiceCreateArgs(opts Options) string {
	args := []string{newEntityKeyExpression(opts)}
	for _, attr := range opts.Attrs {
		if isToManyRelation(attr) {
			continue
		}
		args = append(args, "input."+attr.Name)
	}
	return strings.Join(args, ", ")
}

func appServiceUpdateArgs(opts Options) string {
	args := make([]string, 0, len(opts.Attrs))
	for _, attr := range opts.Attrs {
		if isToManyRelation(attr) {
			continue
		}
		args = append(args, "input."+attr.Name)
	}
	return strings.Join(args, ", ")
}

func efPropertyConfigBlock(opts Options) string {
	var propertyConfig strings.Builder
	for _, attr := range opts.Attrs {
		if isToManyRelation(attr) {
			continue
		}
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

// efRelationshipConfigBlock renders only relationships with explicit metadata;
// missing inverse/join details are rejected before templates are executed.
func efRelationshipConfigBlock(opts Options) string {
	var relationships strings.Builder
	for _, attr := range opts.Attrs {
		switch attr.Relation.Kind {
		case relationReference:
			inverse := attr.Relation.Inverse
			if inverse != "" {
				fmt.Fprintf(
					&relationships,
					"            builder.HasOne(x => x.%s).WithMany(x => x.%s).HasForeignKey(x => x.%s);\n",
					attr.Relation.Navigation,
					inverse,
					attr.Relation.ForeignKey,
				)
			} else {
				fmt.Fprintf(
					&relationships,
					"            builder.HasOne(x => x.%s).WithMany().HasForeignKey(x => x.%s);\n",
					attr.Relation.Navigation,
					attr.Relation.ForeignKey,
				)
			}
		case relationCollection:
			fmt.Fprintf(&relationships, "            builder.HasMany(x => x.%s).WithOne(x => x.%s);\n", attr.Name, attr.Relation.Inverse)
		}
	}
	return relationships.String()
}

// assignmentExpression centralizes generated entity setter validation so the
// constructor and Update method keep using the same domain invariant path.
func assignmentExpression(attr Attribute) string {
	name := lowerFirst(attr.Name)
	if isCSharpString(attr.Type) {
		if attr.Required {
			if attr.MaxLength > 0 {
				return fmt.Sprintf("Check.NotNullOrWhiteSpace(%s, nameof(%s), maxLength: %d)", name, name, attr.MaxLength)
			}
			return fmt.Sprintf("Check.NotNullOrWhiteSpace(%s, nameof(%s))", name, name)
		}
		if attr.MaxLength > 0 {
			return fmt.Sprintf("Check.Length(%s, nameof(%s), maxLength: %d)", name, name, attr.MaxLength)
		}
	}
	return name
}

func isFilterableAttribute(attr Attribute) bool {
	if isToManyRelation(attr) {
		return false
	}
	if attr.Filterable != nil {
		return *attr.Filterable
	}
	return isSimpleFilterType(attr.Type)
}

// isToManyRelation identifies relation shapes that are navigation collections
// rather than scalar input/filter fields.
func isToManyRelation(attr Attribute) bool {
	return attr.Relation.Kind == relationCollection ||
		attr.Relation.Kind == relationManyToMany
}

// isSimpleFilterType limits generated exact filters to C# scalar types that do
// not require custom query semantics.
func isSimpleFilterType(typ string) bool {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "string", "guid", "int", "long", "short", "decimal", "double", "float", "bool", "datetime", "datetimeoffset":
		return true
	default:
		return false
	}
}

// isCSharpString keeps string-specific filter and validation decisions in one
// place.
func isCSharpString(typ string) bool {
	return strings.EqualFold(strings.TrimSpace(typ), "string")
}

// nullableFilterType returns the filter DTO property type for exact-match
// filters.
func nullableFilterType(typ string) string {
	if isCSharpString(typ) {
		return "string"
	}
	return strings.TrimSuffix(strings.TrimSpace(typ), "?")
}

// dtoAttributes removes collection navigations from DTO surfaces that should
// only expose scalar values generated from attributes.
func dtoAttributes(attrs []Attribute) []Attribute {
	out := make([]Attribute, 0, len(attrs))
	for _, attr := range attrs {
		if isToManyRelation(attr) {
			continue
		}
		out = append(out, attr)
	}
	return out
}

// firstManyToManyRelation returns the first configured many-to-many relation;
// v0.2 supports one explicit join entity per generated CRUD target.
func firstManyToManyRelation(opts Options) Relation {
	for _, attr := range opts.Attrs {
		if attr.Relation.Kind == relationManyToMany {
			return attr.Relation
		}
	}
	return Relation{}
}

func attributeUsingBlock(attrs []Attribute) string {
	namespaces := map[string]bool{}
	for _, attr := range attrs {
		if isToManyRelation(attr) {
			continue
		}
		if usesGenericCollections(attr.Type) {
			namespaces["System.Collections.Generic"] = true
		}
	}
	for _, attr := range attrs {
		if isToManyRelation(attr) {
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
