// Package constants mirrors com.example.beanvalidationdemo.constants.
package constants

// Sexs is the accepted-sex pattern.
//
// It is declared and never referenced — PersonRequest.sex carries its own
// @Pattern with an equivalent but separately written expression. It is carried
// over as the dead constant it is in the original rather than wired in, because
// wiring it would change what the application does.
const Sexs = "((^Man$|^Woman$|^UGM$))"
