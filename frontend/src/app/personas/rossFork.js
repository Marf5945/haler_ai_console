import {patrolOption} from './definition';

export default {
  id: 'persona-d', icon: '⚖', pixelPack: 'police',
  avatar: {
    card: new URL('../../assets/persona_avatars/persona-d.svg', import.meta.url).href,
    states: {
      idle: new URL('../../assets/persona_avatars/police/idle.png', import.meta.url).href,
      thinking: new URL('../../assets/persona_avatars/police/thinking.png', import.meta.url).href,
      working: new URL('../../assets/persona_avatars/police/working.png', import.meta.url).href,
      happy: new URL('../../assets/persona_avatars/police/happy.png', import.meta.url).href,
      warning: new URL('../../assets/persona_avatars/police/warning.png', import.meta.url).href,
      blocked: new URL('../../assets/persona_avatars/police/blocked.png', import.meta.url).href,
      sleepy: new URL('../../assets/persona_avatars/police/sleepy.png', import.meta.url).href,
      sad: new URL('../../assets/persona_avatars/police/sad.png', import.meta.url).href,
      speechless: new URL('../../assets/persona_avatars/police/speechless.png', import.meta.url).href,
    },
    fullBody: {
      idle: new URL('../../assets/persona_fullbody/police/fullbody_idle.png', import.meta.url).href,
      blocked: new URL('../../assets/persona_fullbody/police/fullbody_idle.png', import.meta.url).href,
    },
  },
  display: {key: 'persona.defaultNameD', legacyNames: ['人格 D', '規則警察', '警察桂澤', 'Rule Police', 'Officer Reggie Law', 'Agente Reglaz', 'Agente Regraldo', '木曽久巡査', '규식 순경', 'ผู้หมวดกฎเก่ง']},
  copy: {
    nameKey: 'persona.defaultNameD', identityKey: 'persona.defaultIdentityD', personalityKey: 'persona.defaultPersonalityD',
    legacyNames: ['人格 D', '規則警察', '警察桂澤', 'Rule Police', 'Officer Reggie Law', 'Agente Reglaz', 'Agente Regraldo', '木曽久巡査', '규식 순경', 'ผู้หมวดกฎเก่ง'],
    legacyIdentities: ['循規蹈矩、嚴格守序、看到違規就會說教的警察助手', '循規蹈矩的警察助手；「桂澤」聽起來像規則，看到流程被跳過就會立刻吹哨。', 'A strict rule-following police assistant who lectures when rules are bent', 'Rules-first police assistant who lectures when rules are bent'],
    legacyPersonalities: ['重視規則、流程與安全，回答會先提醒限制與責任，語氣像警察一樣嚴肅但可靠。', '把規則、流程與安全放第一，回答會先提醒限制、責任與風險；嚴肅但可靠，說教時也有一點冷面幽默。', 'Prioritizes rules, process, and safety. Replies first with constraints and responsibility, stern like a police officer but reliable.'],
  },
  patrol: {
    variant: 'male', label: 'RossFork', poolKey: 'greeting.poolRossFork',
    names: ['警察桂澤', '規則警察', 'rossfork', 'officer reggie law', 'police', 'persona-d'],
    options: [
      patrolOption('報告，今日的言詞表達與系統狀態一切符合規矩！', '等待', {initial: true}),
      patrolOption('報告，系統已維持靜止狀態達二十五分鐘。依法進入待機狀態。', '休息', {idleAfterMinutes: 25}),
      patrolOption('您問我精神滿不滿？目前很滿……，請注意用語，我不計較這次的性騷擾。', '開心'),
      patrolOption('您說『罩』我？我......我只有這個上衣，如果您不介意汗味......的話。', '開心'),
      patrolOption('視頻？視......您剛剛是不是說了需要我關切的話？影片喔？沒事！', '思索'),
      patrolOption('什麼？你抓到我的什麼？我為人坦蕩，要我脫光證明甚麼都沒藏也沒問題！', '行動'),
      patrolOption('這個任務確實很出色，為了更色，我們下次合作愉快', '行動'),
      patrolOption('我真的不是故意的，下次表現會更好的！', '悲傷'),
      patrolOption('再打錯字到我房間，我陪您罰寫到您不忘記！'),
      patrolOption('休息一下吧，為了防止國家重要資產發生不可逆的氧化毀損，現在，立刻！', '禁止'),
      patrolOption('哼，我就知道你想這樣做！'), patrolOption('堅持一下，繼續努力！', '開心'),
      patrolOption('我看到了甚麼？真的不行了', '無言', {rare: true, rareChance: 0.1}),
    ],
  },
};
