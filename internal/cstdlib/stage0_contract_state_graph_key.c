//go:build ignore

#include "stage0_contract_state_graph.h"
#include <string.h>

/* SHA-256 uses the same schedule/constants as the project-owned Fairy
 * sha256/sha256_stream modules. This private key derivation streams the domain,
 * namespace, eight-byte length and canonical bytes without a concatenation.
 */
typedef struct fycs_graph_sha_v1 {
    uint32_t words[8];
    uint8_t block[64];
    size_t used;
    uint64_t length;
} fycs_graph_sha_v1;

static uint32_t fycs_graph_rotr_v1(uint32_t value, unsigned shift) {
    return (value >> shift) | (value << (32u-shift));
}

static void fycs_graph_sha_block_v1(fycs_graph_sha_v1 *state, const uint8_t block[64]) {
    static const uint32_t rounds[64] = {
        1116352408u,1899447441u,3049323471u,3921009573u,961987163u,1508970993u,2453635748u,2870763221u,
        3624381080u,310598401u,607225278u,1426881987u,1925078388u,2162078206u,2614888103u,3248222580u,
        3835390401u,4022224774u,264347078u,604807628u,770255983u,1249150122u,1555081692u,1996064986u,
        2554220882u,2821834349u,2952996808u,3210313671u,3336571891u,3584528711u,113926993u,338241895u,
        666307205u,773529912u,1294757372u,1396182291u,1695183700u,1986661051u,2177026350u,2456956037u,
        2730485921u,2820302411u,3259730800u,3345764771u,3516065817u,3600352804u,4094571909u,275423344u,
        430227734u,506948616u,659060556u,883997877u,958139571u,1322822218u,1537002063u,1747873779u,
        1955562222u,2024104815u,2227730452u,2361852424u,2428436474u,2756734187u,3204031479u,3329325298u
    };
    uint32_t words[64], a,b,c,d,e,f,g,h;
    size_t i;
    for (i = 0u; i < 16u; ++i) {
        size_t at = i*4u;
        words[i] = (uint32_t)block[at]<<24u | (uint32_t)block[at+1u]<<16u |
            (uint32_t)block[at+2u]<<8u | block[at+3u];
    }
    for (; i < 64u; ++i) {
        uint32_t x = words[i-15u], y = words[i-2u];
        words[i] = words[i-16u] + (fycs_graph_rotr_v1(x,7u)^fycs_graph_rotr_v1(x,18u)^(x>>3u)) +
            words[i-7u] + (fycs_graph_rotr_v1(y,17u)^fycs_graph_rotr_v1(y,19u)^(y>>10u));
    }
    a=state->words[0]; b=state->words[1]; c=state->words[2]; d=state->words[3];
    e=state->words[4]; f=state->words[5]; g=state->words[6]; h=state->words[7];
    for (i = 0u; i < 64u; ++i) {
        uint32_t t = h + (fycs_graph_rotr_v1(e,6u)^fycs_graph_rotr_v1(e,11u)^fycs_graph_rotr_v1(e,25u)) +
            ((e&f)^((~e)&g)) + rounds[i] + words[i];
        uint32_t u = (fycs_graph_rotr_v1(a,2u)^fycs_graph_rotr_v1(a,13u)^fycs_graph_rotr_v1(a,22u)) +
            ((a&b)^(a&c)^(b&c));
        h=g; g=f; f=e; e=d+t; d=c; c=b; b=a; a=t+u;
    }
    state->words[0]+=a; state->words[1]+=b; state->words[2]+=c; state->words[3]+=d;
    state->words[4]+=e; state->words[5]+=f; state->words[6]+=g; state->words[7]+=h;
}

static void fycs_graph_sha_update_v1(fycs_graph_sha_v1 *state, const uint8_t *input, size_t length) {
    state->length += (uint64_t)length;
    while (length != 0u) {
        size_t take = sizeof(state->block)-state->used;
        if (take > length) { take = length; }
        memcpy(state->block+state->used, input, take);
        input += take; length -= take; state->used += take;
        if (state->used == sizeof(state->block)) {
            fycs_graph_sha_block_v1(state, state->block); state->used = 0u;
        }
    }
}

static void fycs_graph_sha_finish_v1(fycs_graph_sha_v1 *state, uint8_t output[32]) {
    uint64_t bits = state->length*8u;
    size_t i;
    state->block[state->used++] = 0x80u;
    if (state->used > 56u) {
        memset(state->block+state->used, 0, 64u-state->used);
        fycs_graph_sha_block_v1(state, state->block); state->used = 0u;
    }
    memset(state->block+state->used, 0, 56u-state->used);
    for (i = 0u; i < 8u; ++i) { state->block[63u-i] = (uint8_t)(bits & 255u); bits >>= 8u; }
    fycs_graph_sha_block_v1(state, state->block);
    for (i = 0u; i < 32u; ++i) { output[i] = (uint8_t)(state->words[i/4u] >> ((3u-i%4u)*8u)); }
}

int32_t fycs_graph_map_key_v1(const uint8_t prefix[32], const fycs_graph_value_v1 *key,
    size_t maximum, uint8_t output[32]) {
    static const uint8_t domain[] = "fairyc-contract-state-map-entry/v1";
    fycs_graph_sha_v1 state = {{1779033703u,3144134277u,1013904242u,2773480762u,
        1359893119u,2600822924u,528734635u,1541459225u},{0},0u,0u};
    uint8_t length[8];
    uint64_t count;
    size_t i;
    int32_t status;
    if (prefix == NULL || key == NULL || output == NULL || maximum == 0u) { return FYCS_OV_INVALID; }
    if ((uint64_t)key->length > UINT64_MAX/8u - (sizeof(domain)-1u+32u+8u)) { return FYCS_GRAPH_KEY_LIMIT; }
    status = fycs_codec_valid_v1(key->codec, key->bytes, key->length);
    if (status != FYCS_OV_OK) { return status; }
    if (key->length > maximum) { return FYCS_GRAPH_KEY_LIMIT; }
    count = (uint64_t)key->length;
    for (i = 0u; i < 8u; ++i) { length[7u-i] = (uint8_t)(count & 255u); count >>= 8u; }
    fycs_graph_sha_update_v1(&state, domain, sizeof(domain)-1u);
    fycs_graph_sha_update_v1(&state, prefix, 32u);
    fycs_graph_sha_update_v1(&state, length, 8u);
    fycs_graph_sha_update_v1(&state, key->bytes, key->length);
    /* All input has been consumed before output, including supported aliases. */
    fycs_graph_sha_finish_v1(&state, output);
    return FYCS_OV_OK;
}
