import {
  EventsOn as WailsEventsOn,
} from '../../wailsjs/runtime/runtime';

function hasWailsRuntime() {
  return typeof window !== 'undefined' && Boolean(window.runtime);
}

export function EventsOn(eventName, callback) {
  if (!hasWailsRuntime()) return () => {};
  try {
    return WailsEventsOn(eventName, callback);
  } catch (error) {
    console.warn('[wailsRuntime] EventsOn unavailable outside Wails runtime', eventName, error);
    return () => {};
  }
}

