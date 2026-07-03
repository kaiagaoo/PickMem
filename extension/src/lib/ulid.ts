// ULID generator matching the Go side's vault.NewID (oklog/ulid). The CLI
// rejects notes whose id doesn't pass ulid.ParseStrict, so captured notes
// must carry a real ULID: 26 Crockford-base32 chars, 48-bit millisecond
// timestamp + 80 random bits.

const ENCODING = "0123456789ABCDEFGHJKMNPQRSTVWXYZ";

export function newULID(now: number = Date.now()): string {
  // Time component: 10 chars, most significant first. 48 bits fits well
  // inside Number's 53-bit integer range, so plain math is exact.
  let t = Math.floor(now);
  const time = new Array<string>(10);
  for (let i = 9; i >= 0; i--) {
    time[i] = ENCODING[t % 32]!;
    t = Math.floor(t / 32);
  }

  // Random component: 16 chars of 5 bits each = 80 random bits.
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  let rand = "";
  for (let i = 0; i < 16; i++) {
    rand += ENCODING[bytes[i]! & 31]!;
  }

  return time.join("") + rand;
}
