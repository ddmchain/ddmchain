
#include "io.h"
#include <string.h>
#include <stdio.h>
#include <errno.h>

enum ddmhash_io_rc ddmhash_io_prepare(
	char const* dirname,
	ddmhash_h256_t const seedhash,
	FILE** output_file,
	uint64_t file_size,
	bool force_create
)
{
	char mutable_name[DAG_MUTABLE_NAME_MAX_SIZE];
	enum ddmhash_io_rc ret = DDMHASH_IO_FAIL;

	errno = 0;

	if (!ddmhash_mkdir(dirname)) {
		DDMHASH_CRITICAL("Could not create the ddmhash directory");
		goto end;
	}

	ddmhash_io_mutable_name(DDMHASH_REVISION, &seedhash, mutable_name);
	char* tmpfile = ddmhash_io_create_filename(dirname, mutable_name, strlen(mutable_name));
	if (!tmpfile) {
		DDMHASH_CRITICAL("Could not create the full DAG pathname");
		goto end;
	}

	FILE *f;
	if (!force_create) {

		f = ddmhash_fopen(tmpfile, "rb+");
		if (f) {
			size_t found_size;
			if (!ddmhash_file_size(f, &found_size)) {
				fclose(f);
				DDMHASH_CRITICAL("Could not query size of DAG file: \"%s\"", tmpfile);
				goto free_memo;
			}
			if (file_size != found_size - DDMHASH_DAG_MAGIC_NUM_SIZE) {
				fclose(f);
				ret = DDMHASH_IO_MEMO_SIZE_MISMATCH;
				goto free_memo;
			}

			uint64_t magic_num;
			if (fread(&magic_num, DDMHASH_DAG_MAGIC_NUM_SIZE, 1, f) != 1) {

				fclose(f);
				DDMHASH_CRITICAL("Could not read from DAG file: \"%s\"", tmpfile);
				ret = DDMHASH_IO_MEMO_SIZE_MISMATCH;
				goto free_memo;
			}
			if (magic_num != DDMHASH_DAG_MAGIC_NUM) {
				fclose(f);
				ret = DDMHASH_IO_MEMO_SIZE_MISMATCH;
				goto free_memo;
			}
			ret = DDMHASH_IO_MEMO_MATCH;
			goto set_file;
		}
	}

	f = ddmhash_fopen(tmpfile, "wb+");
	if (!f) {
		DDMHASH_CRITICAL("Could not create DAG file: \"%s\"", tmpfile);
		goto free_memo;
	}

	if (fseek(f, (long int)(file_size + DDMHASH_DAG_MAGIC_NUM_SIZE - 1), SEEK_SET) != 0) {
		fclose(f);
		DDMHASH_CRITICAL("Could not seek to the end of DAG file: \"%s\". Insufficient space?", tmpfile);
		goto free_memo;
	}
	if (fputc('\n', f) == EOF) {
		fclose(f);
		DDMHASH_CRITICAL("Could not write in the end of DAG file: \"%s\". Insufficient space?", tmpfile);
		goto free_memo;
	}
	if (fflush(f) != 0) {
		fclose(f);
		DDMHASH_CRITICAL("Could not flush at end of DAG file: \"%s\". Insufficient space?", tmpfile);
		goto free_memo;
	}
	ret = DDMHASH_IO_MEMO_MISMATCH;
	goto set_file;

	ret = DDMHASH_IO_MEMO_MATCH;
set_file:
	*output_file = f;
free_memo:
	free(tmpfile);
end:
	return ret;
}
