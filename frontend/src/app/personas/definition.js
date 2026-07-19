const EXPRESSION_ALIASES = {
  '等待': 'idle', '待機': 'idle', '休息': 'sleepy', '睡眠': 'sleepy',
  '行動': 'working', '工作': 'working', '開心': 'happy', '快樂': 'happy',
  '思索': 'thinking', '思考': 'thinking', '悲傷': 'sad', '難過': 'sad',
  '禁止': 'blocked', '阻擋': 'blocked', '警告': 'warning', '注意': 'warning',
  '無言': 'speechless', '傻眼': 'speechless',
  idle: 'idle', waiting: 'idle', sleepy: 'sleepy', rest: 'sleepy',
  working: 'working', action: 'working', happy: 'happy', thinking: 'thinking',
  sad: 'sad', blocked: 'blocked', warning: 'warning', speechless: 'speechless',
};

export function patrolOption(text, expression = 'idle', extra = {}) {
  return {
    text,
    expression: EXPRESSION_ALIASES[String(expression || '').trim()] || 'idle',
    ...extra,
  };
}

export function createPersonaSeed(definition, translate) {
  return {
    id: definition.id,
    name: translate(definition.copy.nameKey),
    icon: definition.icon,
    avatarUrl: '',
    identity: translate(definition.copy.identityKey),
    replyStrategy: '',
    roleStrength: '20%',
    personality: definition.copy.personalityKey ? translate(definition.copy.personalityKey) : '',
    scenario: '',
    description: '',
    patrolDialogue: '',
  };
}
