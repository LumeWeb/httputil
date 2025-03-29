package httputil

import "github.com/getkin/kin-openapi/openapi3"

type apiInfo struct {
	title       string
	version     string
	description string
	contact     *openapi3.Contact
	license     *openapi3.License
	terms       string
}

func APIInfo() *apiInfo {
	return &apiInfo{}
}

func (i *apiInfo) Title(title string) *apiInfo {
	i.title = title
	return i
}

func (i *apiInfo) Version(version string) *apiInfo {
	i.version = version
	return i
}

func (i *apiInfo) Description(desc string) *apiInfo {
	i.description = desc
	return i
}

func (i *apiInfo) Contact(email, name string) *apiInfo {
	i.contact = &openapi3.Contact{
		Email: email,
		Name:  name,
	}
	return i
}

func (i *apiInfo) License(name, url string) *apiInfo {
	i.license = &openapi3.License{
		Name: name,
		URL:  url,
	}
	return i
}

func (i *apiInfo) TermsOfService(terms string) *apiInfo {
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
