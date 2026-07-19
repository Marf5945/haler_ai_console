import youle from './youle';
import grumpyUncle from './grumpyUncle';
import assiStand from './assiStand';
import rossFork from './rossFork';
import touharuMiko from './touharuMiko';
import {createPersonaSeed} from './definition';

export const BUILT_IN_PERSONAS = [youle, grumpyUncle, assiStand, rossFork, touharuMiko];
export const LOCKED_PERSONA_ID = youle.id;
export const BUILT_IN_PERSONA_BY_ID = Object.fromEntries(BUILT_IN_PERSONAS.map((definition) => [definition.id, definition]));
export const PATROL_DIALOGUE_ROLE_VARIANTS = BUILT_IN_PERSONAS.map((definition) => definition.patrol);
export const PERSONA_AVATAR_URLS = Object.fromEntries(BUILT_IN_PERSONAS.map((definition) => [definition.id, definition.avatar.card]));
export const PERSONA_STATE_AVATAR_URLS = Object.fromEntries(BUILT_IN_PERSONAS.map((definition) => [definition.pixelPack, definition.avatar.states]));
export const PERSONA_FULL_BODY_URLS = Object.fromEntries(BUILT_IN_PERSONAS.map((definition) => [definition.pixelPack, definition.avatar.fullBody]));
export const DEFAULT_PERSONA_DISPLAY_NAMES = Object.fromEntries(BUILT_IN_PERSONAS.map((definition) => [definition.id, definition.display]));

export function createBuiltInPersonaSeeds(translate) {
  return BUILT_IN_PERSONAS.map((definition) => createPersonaSeed(definition, translate));
}

export function defaultPixelPackForPersona(personaId) {
  return BUILT_IN_PERSONA_BY_ID[personaId]?.pixelPack || youle.pixelPack;
}
