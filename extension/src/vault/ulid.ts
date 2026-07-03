// A minimal ULID generator producing ids byte-compatible with the Go
// side's github.com/oklog/ulid (26-char Crockford base32, 48-bit
// millisecond timestamp + 80 bits of randomness). The Go store validates
// every note's id with ulid.ParseStrict on load, so the format here must
// match exactly.

// Crockford base32 alphabet — no I, L, O, or U.
const ENC = "0123456789ABCDEFGHJKMNPQRSTVWXYZ";

function encodeTime(ms: number, len = 10): string {
  let out = "";
  for (let i = len - 1; i >= 0; i--) {
    out = ENC[ms % 32] + out;
    ms = Math.floor(ms / 32);
  }
  return out;
}

function encodeRandom(len = 16): string {
  const bytes = crypto.getRandomValues(new Uint8Array(len));
  let out = "";
  for (let i = 0; i < len; i++) {
    out += ENC[bytes[i]! % 32]; // 5-bit mask; every result is a valid symbol
  }
  return out;
}

// ulid returns a fresh 26-character ULID string.
export function ulid(): string {
  return encodeTime(Date.now()) + encodeRandom();
}
