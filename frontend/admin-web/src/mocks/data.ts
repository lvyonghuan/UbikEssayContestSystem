import type {
  Author,
  Contest,
  FlowMount,
  FlowStep,
  ScriptDefinition,
  ScriptFlow,
  ScriptVersion,
  SubAdminInfo,
  Track,
  Work,
} from '@/types/api'

export const mockTokens = {
  access_token: 'mock_access_token',
  refresh_token: 'mock_refresh_token',
}

export const mockContests: Contest[] = [
  {
    contestID: 1,
    contestName: '2026 青年科幻征文',
    contestIntroduction: '面向高校与青年作者的年度征文活动。',
    contestStartDate: '2026-04-01 09:00',
    contestEndDate: '2026-08-01 18:00',
  },
  {
    contestID: 2,
    contestName: '短篇推理创作计划',
    contestIntroduction: '短篇推理赛道与评审流程实验。',
    contestStartDate: '2026-05-10 09:00',
    contestEndDate: '2026-09-30 18:00',
  },
]

export const mockTracksByContest: Record<number, Track[]> = {
  1: [
    {
      trackID: 101,
      contestID: 1,
      trackName: '硬核科幻',
      trackDescription: '强调科学设定与逻辑推演。',
      trackSettings: { reviewMode: 'double-blind' },
    },
    {
      trackID: 102,
      contestID: 1,
      trackName: '太空歌剧',
      trackDescription: '强调叙事节奏与世界观。',
      trackSettings: { reviewMode: 'blind' },
    },
  ],
  2: [
    {
      trackID: 201,
      contestID: 2,
      trackName: '密室推理',
      trackDescription: '限定场景与线索闭环。',
      trackSettings: { reviewMode: 'double-blind' },
    },
  ],
}

export const mockScripts: ScriptDefinition[] = [
  {
    scriptID: 1,
    scriptName: '初稿格式规范检查',
    scriptDescription: '检查标题、字数和基本字段完整性。',
    isEnabled: true,
    activeVersionID: 11,
    extensionData: { runtime: 'python3.11' },
    createdAt: '2026-03-15 10:30',
  },
  {
    scriptID: 2,
    scriptName: '敏感词过滤',
    scriptDescription: '提交前自动扫描敏感词并标记结果。',
    isEnabled: true,
    activeVersionID: 21,
    extensionData: { runtime: 'python3.11' },
    createdAt: '2026-03-20 09:10',
  },
  {
    scriptID: 3,
    scriptName: 'AI 文本提示检测',
    scriptDescription: '辅助检测机器生成痕迹，输出风险分。',
    isEnabled: false,
    activeVersionID: 31,
    extensionData: { runtime: 'python3.11' },
    createdAt: '2026-03-25 13:40',
  },
]

export const mockScriptVersionsByScript: Record<number, ScriptVersion[]> = {
  1: [
    {
      versionID: 11,
      scriptID: 1,
      versionName: 'v1.0.0',
      fileName: 'format_checker.py',
      isActive: true,
      createdAt: '2026-03-15 10:30',
    },
    {
      versionID: 12,
      scriptID: 1,
      versionName: 'v1.1.0',
      fileName: 'format_checker_v1_1.py',
      isActive: false,
      createdAt: '2026-03-29 17:15',
    },
  ],
  2: [
    {
      versionID: 21,
      scriptID: 2,
      versionName: 'v1.0.0',
      fileName: 'sensitive_filter.py',
      isActive: true,
      createdAt: '2026-03-20 09:10',
    },
  ],
  3: [
    {
      versionID: 31,
      scriptID: 3,
      versionName: 'v0.9.0',
      fileName: 'ai_detector.py',
      isActive: true,
      createdAt: '2026-03-25 13:40',
    },
  ],
}

export const mockFlows: ScriptFlow[] = [
  {
    flowID: 1,
    flowName: '投稿预处理流程',
    flowDescription: '投稿后自动完成基础检查与敏感词扫描。',
    isEnabled: true,
    extensionData: { trigger: 'work_created' },
    createdAt: '2026-03-26 11:00',
  },
  {
    flowID: 2,
    flowName: '复审辅助流程',
    flowDescription: '复审阶段调用检测脚本生成风险提醒。',
    isEnabled: false,
    extensionData: { trigger: 'review_assigned' },
    createdAt: '2026-03-30 14:20',
  },
]

export const mockFlowStepsByFlow: Record<number, FlowStep[]> = {
  1: [
    {
      stepID: 101,
      flowID: 1,
      stepOrder: 1,
      stepName: '基础格式检查',
      scriptID: 1,
      stepConfig: { rejectOnError: true },
    },
    {
      stepID: 102,
      flowID: 1,
      stepOrder: 2,
      stepName: '敏感词扫描',
      scriptID: 2,
      stepConfig: { blockLevel: 'medium' },
    },
  ],
  2: [
    {
      stepID: 201,
      flowID: 2,
      stepOrder: 1,
      stepName: 'AI 风险分析',
      scriptID: 3,
      stepConfig: { threshold: 0.8 },
    },
  ],
}

export const mockFlowMountsByFlow: Record<number, FlowMount[]> = {
  1: [
    {
      mountID: 1,
      flowID: 1,
      containerType: 'track',
      containerID: 101,
      mountConfig: { phase: 'submission' },
      createdAt: '2026-03-26 11:30',
    },
    {
      mountID: 2,
      flowID: 1,
      containerType: 'track',
      containerID: 102,
      mountConfig: { phase: 'submission' },
      createdAt: '2026-03-27 09:00',
    },
  ],
  2: [
    {
      mountID: 3,
      flowID: 2,
      containerType: 'contest',
      containerID: 1,
      mountConfig: { phase: 'review' },
      createdAt: '2026-04-01 12:00',
    },
  ],
}

export const mockSubAdmins: SubAdminInfo[] = [
  {
    adminID: 11,
    adminName: '赛事运营-李青',
    adminEmail: 'liqing@ubik.com',
    isActive: true,
    permissionNames: ['contest.read', 'contest.write', 'track.write', 'work.read'],
  },
  {
    adminID: 12,
    adminName: '内容审核-周宁',
    adminEmail: 'zhouning@ubik.com',
    isActive: true,
    permissionNames: ['work.read', 'work.review'],
  },
]

export const mockAuthors: Author[] = [
  {
    authorID: 9001,
    authorName: '陈暮',
    penName: '晨光',
    authorEmail: 'chenmu@example.com',
    authorInfos: {
      city: '上海',
      preferredGenre: 'hard-sci-fi',
    },
  },
  {
    authorID: 9002,
    authorName: '夏洛',
    penName: 'X.L',
    authorEmail: 'xialuo@example.com',
    authorInfos: {
      city: '北京',
      preferredGenre: 'space-opera',
    },
  },
  {
    authorID: 9003,
    authorName: '林叙',
    penName: '叙言',
    authorEmail: 'linxu@example.com',
    authorInfos: {
      city: '广州',
      preferredGenre: 'mystery',
    },
  },
]

export const mockWorksByTrack: Record<number, Work[]> = {
  101: [
    {
      workID: 5001,
      authorID: 9001,
      authorName: '陈暮',
      trackID: 101,
      trackName: '硬核科幻',
      workTitle: '流浪火种',
      workInfos: {
        status: '待审核',
        submittedAt: '2026-04-03 14:10',
        wordCount: 5200,
      },
    },
    {
      workID: 5002,
      authorID: 9002,
      authorName: '夏洛',
      trackID: 101,
      trackName: '硬核科幻',
      workTitle: '重力尽头',
      workInfos: {
        status: '已通过',
        submittedAt: '2026-04-05 10:22',
        wordCount: 6800,
      },
    },
  ],
  102: [
    {
      workID: 5003,
      authorID: 9003,
      authorName: '林叙',
      trackID: 102,
      trackName: '太空歌剧',
      workTitle: '星门巡礼',
      workInfos: {
        status: '待审核',
        submittedAt: '2026-04-06 09:16',
        wordCount: 7100,
      },
    },
  ],
  201: [
    {
      workID: 5004,
      authorID: 9010,
      authorName: '顾川',
      trackID: 201,
      trackName: '密室推理',
      workTitle: '七号密室',
      workInfos: {
        status: '已驳回',
        submittedAt: '2026-04-04 19:45',
        wordCount: 4600,
      },
    },
    {
      workID: 5005,
      authorID: 9011,
      authorName: '宋砚',
      trackID: 201,
      trackName: '密室推理',
      workTitle: '黑伞谋杀案',
      workInfos: {
        status: '待审核',
        submittedAt: '2026-04-07 16:00',
        wordCount: 5400,
      },
    },
  ],
}

export function getAllMockWorks() {
  return Object.values(mockWorksByTrack).flat()
}
