package entity

import (
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/beanvalidation"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/service/groups"
)

// Person carries a single property constrained only under validation groups.
type Person struct {
	Group *string `json:"group"`
}

// SetGroup is Lombok's @Data setter.
func (p *Person) SetGroup(value string) { p.Group = &value }

// GetGroup is Lombok's @Data getter.
func (p *Person) GetGroup() *string { return p.Group }

// Constraints declares Person's group-scoped annotations. Neither belongs to
// the Default group, so validating a Person without naming a group finds
// nothing to check.
func (Person) Constraints() beanvalidation.Constraints {
	return beanvalidation.Constraints{
		beanvalidation.On("group",
			// 当验证组为 DeletePersonGroup 的时候 group 字段不能为空
			beanvalidation.NotNull().WithGroups(groups.DeletePersonGroup),
			// 当验证组为 AddPersonGroup 的时候 group 字段需要为空
			beanvalidation.Null().WithGroups(groups.AddPersonGroup),
		),
	}
}
