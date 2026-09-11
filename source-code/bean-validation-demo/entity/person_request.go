// Package entity mirrors com.example.beanvalidationdemo.entity.
package entity

import (
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/beanvalidation"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/validation"
)

// The four custom messages, quoted from the annotations.
const (
	ClassIDNotNullMessage     = "classId 不能为空"
	NameNotNullMessage        = "name 不能为空"
	SexNotNullMessage         = "sex 不能为空"
	SexPatternMessage         = "sex 值不在可选范围"
	PhoneNumberNotNullMessage = "phoneNumber 不能为空"
	PhoneNumberFormatMessage  = "phoneNumber 格式不正确"
)

// sexPattern is @Pattern's expression on the sex field.
const sexPattern = `(^Man$|^Woman$|^UGM$)`

// nameMaxSize is @Size(max = 33) on the name field. Its minimum is @Size's
// implicit 0, which the default message still interpolates.
const nameMaxSize = 33

// PersonRequest is the @RequestBody the API accepts.
//
// 微信搜 JavaGuide 回复"面试突击"即可免费领取个人原创的 Java 面试手册
//
// Every property is a pointer because every one is a nullable Java String, and
// the difference is observable: an absent `name` is null and trips @NotNull,
// while `"name": ""` is a present empty string that passes both @NotNull and
// @Size. The field order is the declaration order Jackson serialises a POJO in.
type PersonRequest struct {
	ClassID     *string `json:"classId"`
	Name        *string `json:"name"`
	Sex         *string `json:"sex"`
	Region      *string `json:"region"`
	PhoneNumber *string `json:"phoneNumber"`
}

// Constraints declares the annotations PersonRequest's fields carry, in field
// order and, within a field, in annotation order.
func (PersonRequest) Constraints() beanvalidation.Constraints {
	return beanvalidation.Constraints{
		beanvalidation.On("classId",
			beanvalidation.NotNull().WithMessage(ClassIDNotNullMessage)),
		beanvalidation.On("name",
			beanvalidation.Size(0, nameMaxSize),
			beanvalidation.NotNull().WithMessage(NameNotNullMessage)),
		beanvalidation.On("sex",
			beanvalidation.Pattern(sexPattern).WithMessage(SexPatternMessage),
			beanvalidation.NotNull().WithMessage(SexNotNullMessage)),
		beanvalidation.On("region",
			validation.Region()),
		beanvalidation.On("phoneNumber",
			validation.PhoneNumber().WithMessage(PhoneNumberFormatMessage),
			beanvalidation.NotNull().WithMessage(PhoneNumberNotNullMessage)),
	}
}

// PersonRequestBuilder is Lombok's @Builder, which the tests use.
type PersonRequestBuilder struct{ built PersonRequest }

// NewPersonRequestBuilder is PersonRequest.builder().
func NewPersonRequestBuilder() *PersonRequestBuilder { return &PersonRequestBuilder{} }

// ClassID sets classId.
func (b *PersonRequestBuilder) ClassID(value string) *PersonRequestBuilder {
	b.built.ClassID = &value
	return b
}

// Name sets name.
func (b *PersonRequestBuilder) Name(value string) *PersonRequestBuilder {
	b.built.Name = &value
	return b
}

// Sex sets sex.
func (b *PersonRequestBuilder) Sex(value string) *PersonRequestBuilder {
	b.built.Sex = &value
	return b
}

// Region sets region.
func (b *PersonRequestBuilder) Region(value string) *PersonRequestBuilder {
	b.built.Region = &value
	return b
}

// PhoneNumber sets phoneNumber.
func (b *PersonRequestBuilder) PhoneNumber(value string) *PersonRequestBuilder {
	b.built.PhoneNumber = &value
	return b
}

// Build returns the request. Properties never set stay null.
func (b *PersonRequestBuilder) Build() *PersonRequest {
	built := b.built
	return &built
}
