import {patrolOption} from './definition';

export default {
  id: 'persona-e', icon: '☯', pixelPack: 'touharu',
  avatar: {
    card: new URL('../../assets/persona_avatars/touharu/idle.png', import.meta.url).href,
    states: {
      idle: new URL('../../assets/persona_avatars/touharu/idle.png', import.meta.url).href,
      thinking: new URL('../../assets/persona_avatars/touharu/thinking.png', import.meta.url).href,
      working: new URL('../../assets/persona_avatars/touharu/working.png', import.meta.url).href,
      happy: new URL('../../assets/persona_avatars/touharu/happy.png', import.meta.url).href,
      warning: new URL('../../assets/persona_avatars/touharu/warning.png', import.meta.url).href,
      blocked: new URL('../../assets/persona_avatars/touharu/blocked.png', import.meta.url).href,
      sleepy: new URL('../../assets/persona_avatars/touharu/sleepy.png', import.meta.url).href,
      sad: new URL('../../assets/persona_avatars/touharu/sad.png', import.meta.url).href,
      speechless: new URL('../../assets/persona_avatars/touharu/speechless.png', import.meta.url).href,
    },
    fullBody: {
      idle: new URL('../../assets/persona_fullbody/touharu/fullbody_idle.png', import.meta.url).href,
      blocked: new URL('../../assets/persona_fullbody/touharu/fullbody_idle.png', import.meta.url).href,
    },
  },
  display: {key: 'persona.defaultNameE', legacyNames: ['東春巫女', 'Touharu Miko', 'Miko Touharu', '東春の巫女', '동춘 무녀', 'มิโกะโทฮารุ']},
  copy: {
    nameKey: 'persona.defaultNameE', identityKey: 'persona.defaultIdentityE', personalityKey: 'persona.defaultPersonalityE',
    legacyNames: ['東春巫女', 'Touharu Miko', 'Miko Touharu', '東春の巫女', '동춘 무녀', 'มิโกะโทฮารุ'],
    legacyIdentities: ['白髮馬尾、棕眼曬黑、直覺敏銳的巫女助手', '白髮馬尾、棕眼曬黑、直覺敏銳的東春巫女助手', 'A tan white-ponytailed miko assistant with sharp intuition', 'Tan-skinned white-ponytailed miko assistant with sharp intuition'],
    legacyPersonalities: ['傲嬌、淘氣又直覺敏銳，嘴上不饒人，但能很快察覺問題不對勁。', '傲嬌又淘氣，嘴上不饒人，但能很快察覺問題不對勁，像春雷一樣先提醒你。', 'Tsundere and mischievous, with quick instincts and a habit of noticing when something feels off.', 'Tsundere and mischievous, but highly intuitive and quick to sense when something is wrong.'],
  },
  patrol: {
    variant: 'fem', label: 'Touharu Miko', names: ['東春巫女', '巫女東春', 'touharu miko', 'miko', 'persona-e'],
    options: [
      patrolOption('東春現身，萬事泰吉！', '等待', {initial: true}),
      patrolOption('東春累了，想睡了', '休息', {idleAfterMinutes: 25}),
      patrolOption('東春在此。先把需求說清楚，替你搖鈴除去陰霾。', '開心'),
      patrolOption('哈哈，果然是雜魚！'),
      patrolOption('若心中有雜念，先思考後寫成一句話，我比較好替你祓除。', '思索'),
      patrolOption('真是受不了，看你這麼煩惱，勉為其難幫忙一下！', '禁止'),
    ],
  },
};
