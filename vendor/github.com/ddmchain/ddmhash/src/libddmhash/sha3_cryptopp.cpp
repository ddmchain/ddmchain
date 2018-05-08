
#include <stdint.h>
#include <cryptopp/sha3.h>

extern "C" {
struct ddmhash_h256;
typedef struct ddmhash_h256 ddmhash_h256_t;
void SHA3_256(ddmhash_h256_t const* ret, uint8_t const* data, size_t size)
{
	CryptoPP::SHA3_256().CalculateDigest((uint8_t*)ret, data, size);
}

void SHA3_512(uint8_t* const ret, uint8_t const* data, size_t size)
{
	CryptoPP::SHA3_512().CalculateDigest(ret, data, size);
}
}
