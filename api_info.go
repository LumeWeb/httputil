package httputil

import "github.com/getkin/kin-openapi/openapi3"

// APIInfoDefinition defines the contract for building OpenAPI info metadata
type APIInfoDefinition interface {
	Title(title string) APIInfoDefinition
	Version(version string) APIInfoDefinition
	Description(desc string) APIInfoDefinition
	Contact(email, name string) APIInfoDefinition
	License(name, url string) APIInfoDefinition
	TermsOfService(terms string) APIInfoDefinition
	toOpenAPI() *openapi3.Info
}

type apiInfo struct {
	title       string
	version     string
	description string
	contact     *openapi3.Contact
	license     *openapi3.License
	terms       string
}

func APIInfo() APIInfoDefinition {
	return &apiInfo{}
}

func (i *apiInfo) Title(title string) APIInfoDefinition {
	i.title = title
	return i
}

func (i *apiInfo) Version(version string) APIInfoDefinition {
	i.version = version
	return i
}

func (i *apiInfo) Description(desc string) APIInfoDefinition {
	i.description = desc
	return i
}

func (i *apiInfo) Contact(email, name string) APIInfoDefinition {
	i.contact = &openapi3.Contact{
		Email: email,
		Name:  name,
	}
	return i
}

func (i *apiInfo) License(name, url string) APIInfoDefinition {
	i.license = &openapi3.License{
		Name: name,
		URL:  url,
	}
	return i
}

func (i *apiInfo) TermsOfService(terms string) APIInfoDefinition {
	i.terms = terms
	return i
}

func (i *apiInfo) toOpenAPI() *openapi3.Info {
	return &openapi3.Info{
		Title:          i.title,
		Version:        i.version,
		Description:    i.description,
		Contact:        i.contact,
		License:        i.license,
		TermsOfService: i.terms,
	}
}
