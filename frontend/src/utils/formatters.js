/**
 * @file Utility functions for formatting and coercing data types.
 */

/**
 * Converts a date/time string into a Unix timestamp (milliseconds).
 * Returns 0 if the value is invalid.
 * @param {string | null | undefined} value - The date string to parse.
 * @returns {number} The timestamp in milliseconds or 0.
 */
export const toTimestamp = (value) => {
  const time = Date.parse(value ?? '');
  return Number.isNaN(time) ? 0 : time;
};

/**
 * Coerces a value to a positive integer port number.
 * @param {any} value - The value to parse.
 * @param {number} fallback - The fallback value to use if parsing fails.
 * @returns {number} The parsed port number or the fallback.
 */
export const coercePort = (value, fallback) => {
  const parsed = Number.parseInt(value, 10);
  if (Number.isNaN(parsed) || parsed <= 0) {
    return fallback;
  }
  return parsed;
};
