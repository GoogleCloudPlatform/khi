/**
 * Formats a byte count into a human-readable string with binary units (B, KB, MB, GB, TB).
 *
 * @param bytes - Size in bytes.
 * @returns Human-readable formatted string (e.g., "512 B", "1.0 KB", "10 MB").
 */
export function formatBytes(bytes: number): string {
  if (bytes <= 0 || !Number.isFinite(bytes)) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const digitGroup = Math.min(
    Math.floor(Math.log10(bytes) / Math.log10(1024)),
    units.length - 1,
  );
  const value = bytes / Math.pow(1024, digitGroup);
  return `${value.toFixed(value >= 10 || digitGroup === 0 ? 0 : 1)} ${units[digitGroup]}`;
}
