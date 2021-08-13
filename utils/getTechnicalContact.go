package utils

import "strings"

func getTechnicalContactFromAnnotation(contacts string) string {
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
