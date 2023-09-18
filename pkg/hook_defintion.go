package gmakec

import "fmt"

type HookDefinition struct {
	Ref     string             `yaml:"ref"`
	Name    string             `yaml:"name"`
	Step    string             `yaml:"step"`
	Actions []ActionDefinition `yaml:"actions"`
}

func (this *HookDefinition) findRef(refHookDefinitions *[]HookDefinition) *HookDefinition {
	for _, refHookDefinition := range *refHookDefinitions {
		if refHookDefinition.Name == this.Ref {
			return &refHookDefinition
		}
	}

	return nil
}

func (this *HookDefinition) withRef(definitionContext *DefinitionContext) (*HookDefinition, error) {
	if len(this.Ref) == 0 {
		return this, nil
	}

	hookRef := this.findRef(&definitionContext.Definition.Hooks)

	if hookRef == nil {
		return nil, fmt.Errorf("Could not find referenced hook `%s`!", this.Ref)
	}

	for index := range hookRef.Actions {
		action, err := hookRef.Actions[index].withRef(definitionContext)

		if err != nil {
			return nil, err
		}

		hookRef.Actions[index] = *action
	}

	hookRef.Actions = append(hookRef.Actions, this.Actions...)
	return hookRef, nil
}
