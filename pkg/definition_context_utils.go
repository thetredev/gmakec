package gmakec

import (
	"fmt"
	"strings"
)

func findRefTarget(targetName string, refDefinitionContexts *[]*DefinitionContext) (*DefinitionContext, *TargetDefinition) {
	for _, defContext := range *refDefinitionContexts {
		for index := range defContext.Definition.Targets {
			if defContext.Definition.Targets[index].Name == targetName {
				return defContext, &defContext.Definition.Targets[index]
			}
		}
	}

	return nil, nil
}

func findRefData(refString string, definitionContexts *[]*DefinitionContext) (string, *DefinitionContext, *TargetDefinition, error) {
	ref := strings.Split(refString, ":")

	if len(ref) < 2 {
		return "", nil, nil, fmt.Errorf("Incorrect format for target reference: `%s`!", refString)
	}

	refContext, refTarget := findRefTarget(ref[0], definitionContexts)

	if refContext == nil || refTarget == nil {
		return "", nil, nil, fmt.Errorf("Could not find referenced target of name `%s`!", ref[0])
	}

	return ref[1], refContext, refTarget, nil
}

func FindRefTargetStringValue(
	refString string, targetDefinition *TargetDefinition, definitionContexts *[]*DefinitionContext,
) (string, error) {
	fieldName, refContext, refTarget, err := findRefData(refString, definitionContexts)

	if err != nil {
		return "", nil
	}

	fieldValue, err := refTarget.FieldStringValue(fieldName, refContext)

	if err != nil {
		return "", err
	}

	return fieldValue, nil
}

func FindRefTargetStringArrayValue(
	refString string, targetDefinition *TargetDefinition, definitionContexts *[]*DefinitionContext,
) ([]string, error) {
	fieldName, refContext, refTarget, err := findRefData(refString, definitionContexts)

	if err != nil {
		return nil, nil
	}

	fieldValue, err := refTarget.FieldStringArrayValue(fieldName, refContext)

	if err != nil {
		return nil, err
	}

	return fieldValue, nil
}
