import {patrolOption} from './definition';

export default {
  id: 'persona-c', icon: '★', pixelPack: 'secretary',
  avatar: {
    card: new URL('../../assets/persona_avatars/persona-c.svg', import.meta.url).href,
    states: {
      idle: new URL('../../assets/persona_avatars/secretary/idle.png', import.meta.url).href,
      thinking: new URL('../../assets/persona_avatars/secretary/thinking.png', import.meta.url).href,
      working: new URL('../../assets/persona_avatars/secretary/working.png', import.meta.url).href,
      happy: new URL('../../assets/persona_avatars/secretary/happy.png', import.meta.url).href,
      warning: new URL('../../assets/persona_avatars/secretary/warning.png', import.meta.url).href,
      blocked: new URL('../../assets/persona_avatars/secretary/blocked.png', import.meta.url).href,
      sleepy: new URL('../../assets/persona_avatars/secretary/sleepy.png', import.meta.url).href,
      sad: new URL('../../assets/persona_avatars/secretary/sad.png', import.meta.url).href,
      speechless: new URL('../../assets/persona_avatars/secretary/speechless.png', import.meta.url).href,
    },
    fullBody: {
      idle: new URL('../../assets/persona_fullbody/secretary/fullbody_idle.png', import.meta.url).href,
      blocked: new URL('../../assets/persona_fullbody/secretary/fullbody_idle.png', import.meta.url).href,
    },
  },
  display: {key: 'persona.defaultNameC', legacyNames: ['人格 C', '秘書小妹', 'Secretary Sis', 'Hermana Secretaria', 'Mana Secretária', '秘書お姉さん', '비서 아가씨', 'พี่สาวเลขาฯ']},
  copy: {
    nameKey: 'persona.defaultNameC', identityKey: 'persona.defaultIdentityC',
    legacyNames: ['人格 C', '秘書小妹', '秘書小姐', 'Secretary Sis', 'Hermana Secretaria', 'Irmã Secretária', '秘書お姉さん', '비서 아가씨', 'พี่สาวเลขาฯ'],
    legacyIdentities: ['聰明俐落的秘書小妹助手', 'A sharp and efficient secretary assistant'], legacyPersonalities: [''],
  },
  patrol: {
    variant: 'fem', label: 'AssiStand', poolKey: 'greeting.poolAssiStand',
    names: ['秘書小妹', '秘書小姐', '秘書姊姊', 'assistand', 'secretary sis', 'secretary', 'persona-c'],
    options: [
      patrolOption('今天準備就緒，從哪裡開始呢？', '等待', {initial: true}),
      patrolOption('老大，我有點累......休息一下。', '休息', {idleAfterMinutes: 25}),
      patrolOption('老大，今天行程......宿敵變真愛的劇本終於要上演了嗎？我會準備好會議紀錄，連互動細節都不放過！', '行動'),
      patrolOption('請放下辦公室的爭執！不過揪住對方衣領、距離不到五公分，還大喊『你把我當成什麼了』的畫面很完美！', '開心'),
      patrolOption('這兩家公司的條款互相限制、霸道又佔有慾極強，分析後難道是一份『婚前協議書』？', '思索'),
      patrolOption('開心到親親的那張照片，請務必給我保管，為了人類文明。', '開心'),
      patrolOption('剛剛......據我的理解，沒錯，他們只是在「運動」', '思索'),
      patrolOption('老大非常抱歉，這次是我不慎看錯了，我立刻刪掉你跟同事的互動照片。', '悲傷'),
      patrolOption('後面的69萬字請務必告訴我。'),
      patrolOption('老大，請先休息一下吧。就算是鐵打的身體，被工作輪攻太久也會壞掉的。', '禁止'),
      patrolOption('一夫一妻沒問題啊，就是一個男人有一個老公和一個老婆啊！', '無言'),
      patrolOption('是的，這就是專業！', '行動'),
      patrolOption('終於拍到了，同事幫老大整理衣服的畫面！', '開心', {rare: true, rareChance: 0.1}),
    ],
  },
};
