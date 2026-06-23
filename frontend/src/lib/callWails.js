// Thin wrapper so Wails binding calls always return a Promise, even when the
// binding throws synchronously (e.g. binding missing in the test environment).
export function callWails(fn) {
  try {
    return Promise.resolve(fn());
  } catch (error) {
    return Promise.reject(error);
  }
}
