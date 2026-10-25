/**
 * 32-bit FNV-1a non-cryptographic hash function.
 * Matches Go's standard library `hash/fnv` fnv.New32a() exactly.
 */
export function fnv1a(key: string): number {
  let hash = 2166136261; // FNV-1a 32-bit offset basis (0x811c9dc5)
  for (let i = 0; i < key.length; i++) {
    hash ^= key.charCodeAt(i);
    // Multiply by 32-bit FNV prime (16777619 = 0x01000193) and keep as unsigned 32-bit
    hash = Math.imul(hash, 16777619) >>> 0;
  }
  return hash >>> 0;
}

/**
 * Format uint32 hash as a clean 8-character hex string (e.g. "0x8f3a41b2").
 */
export function formatHashHex(hash: number): string {
  return '0x' + (hash >>> 0).toString(16).padStart(8, '0').toUpperCase();
}

/**
 * Convert uint32 hash [0, 2^32 - 1] to angle on circle [0, 360) degrees.
 */
export function hashToAngle(hash: number): number {
  return ((hash >>> 0) / 0xffffffff) * 360;
}
