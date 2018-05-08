
#include "io.h"
#include <sys/types.h>
#include <sys/stat.h>
#include <errno.h>
#include <libgen.h>
#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>
#include <pwd.h>

FILE* ddmhash_fopen(char const* file_name, char const* mode)
{
	return fopen(file_name, mode);
}

char* ddmhash_strncat(char* dest, size_t dest_size, char const* src, size_t count)
{
	return strlen(dest) + count + 1 <= dest_size ? strncat(dest, src, count) : NULL;
}

bool ddmhash_mkdir(char const* dirname)
{
	int rc = mkdir(dirname, S_IRWXU | S_IRWXG | S_IROTH | S_IXOTH);
	return rc != -1 || errno == EEXIST;
}

int ddmhash_fileno(FILE *f)
{
	return fileno(f);
}

char* ddmhash_io_create_filename(
	char const* dirname,
	char const* filename,
	size_t filename_length
)
{
	size_t dirlen = strlen(dirname);
	size_t dest_size = dirlen + filename_length + 1;
	if (dirname[dirlen] != '/') {
		dest_size += 1;
	}
	char* name = malloc(dest_size);
	if (!name) {
		return NULL;
	}

	name[0] = '\0';
	ddmhash_strncat(name, dest_size, dirname, dirlen);
	if (dirname[dirlen] != '/') {
		ddmhash_strncat(name, dest_size, "/", 1);
	}
	ddmhash_strncat(name, dest_size, filename, filename_length);
	return name;
}

bool ddmhash_file_size(FILE* f, size_t* ret_size)
{
	struct stat st;
	int fd;
	if ((fd = fileno(f)) == -1 || fstat(fd, &st) != 0) {
		return false;
	}
	*ret_size = st.st_size;
	return true;
}

bool ddmhash_get_default_dirname(char* strbuf, size_t buffsize)
{
	static const char dir_suffix[] = ".ddmhash/";
	strbuf[0] = '\0';
	char* home_dir = getenv("HOME");
	if (!home_dir || strlen(home_dir) == 0)
	{
		struct passwd* pwd = getpwuid(getuid());
		if (pwd)
			home_dir = pwd->pw_dir;
	}

	size_t len = strlen(home_dir);
	if (!ddmhash_strncat(strbuf, buffsize, home_dir, len)) {
		return false;
	}
	if (home_dir[len] != '/') {
		if (!ddmhash_strncat(strbuf, buffsize, "/", 1)) {
			return false;
		}
	}
	return ddmhash_strncat(strbuf, buffsize, dir_suffix, sizeof(dir_suffix));
}
