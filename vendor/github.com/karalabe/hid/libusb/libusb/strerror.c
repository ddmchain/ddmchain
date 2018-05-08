
#include <config.h>

#include <locale.h>
#include <stdlib.h>
#include <string.h>
#if defined(HAVE_STRINGS_H)
#include <strings.h>
#endif

#include "libusbi.h"

#if defined(_MSC_VER)
#define strncasecmp _strnicmp
#endif

static size_t usbi_locale = 0;

static const char* usbi_locale_supported[] = { "en", "nl", "fr", "ru" };
static const char* usbi_localized_errors[ARRAYSIZE(usbi_locale_supported)][LIBUSB_ERROR_COUNT] = {
	{ 
		"Success",
		"Input/Output Error",
		"Invalid parameter",
		"Access denied (insufficient permissions)",
		"No such device (it may have been disconnected)",
		"Entity not found",
		"Resource busy",
		"Operation timed out",
		"Overflow",
		"Pipe error",
		"System call interrupted (perhaps due to signal)",
		"Insufficient memory",
		"Operation not supported or unimplemented on this platform",
		"Other error",
	}, { 
		"Gelukt",
		"Invoer-/uitvoerfout",
		"Ongeldig argument",
		"Toegang geweigerd (onvoldoende toegangsrechten)",
		"Apparaat bestaat niet (verbinding met apparaat verbroken?)",
		"Niet gevonden",
		"Apparaat of hulpbron is bezig",
		"Bewerking verlopen",
		"Waarde is te groot",
		"Gebroken pijp",
		"Onderbroken systeemaanroep",
		"Onvoldoende geheugen beschikbaar",
		"Bewerking wordt niet ondersteund",
		"Andere fout",
	}, { 
		"Succès",
		"Erreur d'entrée/sortie",
		"Paramètre invalide",
		"Accès refusé (permissions insuffisantes)",
		"Périphérique introuvable (peut-être déconnecté)",
		"Elément introuvable",
		"Resource déjà occupée",
		"Operation expirée",
		"Débordement",
		"Erreur de pipe",
		"Appel système abandonné (peut-être à cause d’un signal)",
		"Mémoire insuffisante",
		"Opération non supportée or non implémentée sur cette plateforme",
		"Autre erreur",
	}, { 
		"Успех",
		"Ошибка ввода/вывода",
		"Неверный параметр",
		"Доступ запрещён (не хватает прав)",
		"Устройство отсутствует (возможно, оно было отсоединено)",
		"Элемент не найден",
		"Ресурс занят",
		"Истекло время ожидания операции",
		"Переполнение",
		"Ошибка канала",
		"Системный вызов прерван (возможно, сигналом)",
		"Память исчерпана",
		"Операция не поддерживается данной платформой",
		"Неизвестная ошибка"
	}
};

int API_EXPORTED libusb_setlocale(const char *locale)
{
	size_t i;

	if ( (locale == NULL) || (strlen(locale) < 2)
	  || ((strlen(locale) > 2) && (locale[2] != '-') && (locale[2] != '_') && (locale[2] != '.')) )
		return LIBUSB_ERROR_INVALID_PARAM;

	for (i=0; i<ARRAYSIZE(usbi_locale_supported); i++) {
		if (strncasecmp(usbi_locale_supported[i], locale, 2) == 0)
			break;
	}
	if (i >= ARRAYSIZE(usbi_locale_supported)) {
		return LIBUSB_ERROR_NOT_FOUND;
	}

	usbi_locale = i;

	return LIBUSB_SUCCESS;
}

DEFAULT_VISIBILITY const char* LIBUSB_CALL libusb_strerror(enum libusb_error errcode)
{
	int errcode_index = -errcode;

	if ((errcode_index < 0) || (errcode_index >= LIBUSB_ERROR_COUNT)) {

		errcode_index = LIBUSB_ERROR_COUNT - 1;
	}

	return usbi_localized_errors[usbi_locale][errcode_index];
}
