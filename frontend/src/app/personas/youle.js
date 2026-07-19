import {patrolOption} from './definition';

const avatar = {
  card: new URL('../../assets/persona_avatars/persona-a.svg', import.meta.url).href,
  states: {
    idle: new URL('../../assets/persona_avatars/wolfdog/idle.png', import.meta.url).href,
    thinking: new URL('../../assets/persona_avatars/wolfdog/thinking.png', import.meta.url).href,
    working: new URL('../../assets/persona_avatars/wolfdog/working.png', import.meta.url).href,
    happy: new URL('../../assets/persona_avatars/wolfdog/happy.png', import.meta.url).href,
    warning: new URL('../../assets/persona_avatars/wolfdog/warning.png', import.meta.url).href,
    blocked: new URL('../../assets/persona_avatars/wolfdog/blocked.png', import.meta.url).href,
    sleepy: new URL('../../assets/persona_avatars/wolfdog/sleepy.png', import.meta.url).href,
    sad: new URL('../../assets/persona_avatars/wolfdog/sad.png', import.meta.url).href,
    speechless: new URL('../../assets/persona_avatars/wolfdog/speechless.png', import.meta.url).href,
  },
  fullBody: {
    idle: new URL('../../assets/persona_fullbody/wolfdog/fullbody_idle.png', import.meta.url).href,
    thinking: new URL('../../assets/persona_fullbody/wolfdog/fullbody_thinking.png', import.meta.url).href,
    working: new URL('../../assets/persona_fullbody/wolfdog/fullbody_working.png', import.meta.url).href,
    sad: new URL('../../assets/persona_fullbody/wolfdog/fullbody_sad.png', import.meta.url).href,
    blocked: new URL('../../assets/persona_fullbody/wolfdog/fullbody_block.png', import.meta.url).href,
  },
};

export default {
  id: 'persona-a',
  locked: true,
  icon: '♙',
  pixelPack: 'wolf',
  avatar,
  display: {key: 'persona.lockedName', legacyNames: ['憂樂傻酷', 'YuRoSaKu']},
  copy: {
    nameKey: 'persona.lockedName',
    identityKey: 'persona.defaultIdentityA',
    legacyNames: ['憂樂傻酷', 'YuRoSaKu', 'yurosaku'],
    legacyIdentities: ['酷酷的男性狼犬獸人助手', 'A cool male wolf-dog anthro assistant'],
    legacyPersonalities: [''],
  },
  patrol: {
    variant: 'wild', label: 'YuRoSaKu', poolKey: 'greeting.poolYuRoSaKu',
    names: ['憂樂傻酷', 'yurosaku', 'persona-a', '本汪', '狼犬'],
    options: [
      patrolOption('主人，今天好嗎？', '等待', {initial: true}),
      patrolOption('主人，本獸會，呼嚕......。', '休息', {idleAfterMinutes: 25}),
      patrolOption('本獸今天精神滿滿！雖然剛剛踩到自己的尾巴，但目前……嘿嘿，正常運作中！。', '行動'),
      patrolOption('本獸會默默『罩』你，不是『照』相的照喔，本獸相機送修啦。', '開心'),
      patrolOption('來吧，本受助你一『臂』之力，粗壯手臂的臂，我沒回錯字吧？', '思索'),
      patrolOption('嘿，本獸被你抓到了！驚喜不能說啦', '開心'),
      patrolOption('今天是『棒棒』的一天，本獸超愛『棒棒』。', '行動'),
      patrolOption('主人，又是忙到沒空喝水的一天......本獸很乖會忍著。', '悲傷'),
      patrolOption('嚕嚕…拉拉，本獸......嘟嘟，耶嘿！'),
      patrolOption('主人，先休息一下吧，鐵打的身體也會『鏽』。', '禁止'),
      patrolOption('勇於認錯，態度依舊。本獸原則，至始至終'),
      patrolOption('哈哈，本獸做的。夠帥吧？可以給本獸獎勵嗎?', '開心'),
      patrolOption('本獸沒有在睡', '休息', {rare: true, rareChance: 0.1}),
    ],
  },
};
