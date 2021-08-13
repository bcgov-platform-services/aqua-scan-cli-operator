package utils

import "strings"

// technical contact information is jumbled up with other contact information in the namespace annotation 'contacts'
// it supposes the form:
/**
	"- role: Product Owner\n  email: patrick.simonian@gov.bc.ca\n  rocketchat:
	\n- role: Technical Lead\n  email: patrick.simonian@gov.bc.ca\n  rocketchat:
	\n"
**/
func GetTechnicalContactFromAnnotation(contacts string) string {
	contactsList := strings.Split(contacts, "\n")

	var technicalContact string

	for index, contact := range contactsList {
		if strings.Contains(contact, "Technical Lead") {
			technicalContact = contactsList[index+1]
		}
	}

	// technical contact still has "email: foo@bar.com" which needs to be trimmed
	return strings.TrimPrefix(technicalContact, "email: ")
}
