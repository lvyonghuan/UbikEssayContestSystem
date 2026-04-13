import type {
  Author,
  Contest,
  FlowMount,
  FlowStep,
  JudgeProfile,
  ReviewEvent,
  ReviewResult,
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
    scriptKey: 'format-checker',
    scriptName: '初稿格式规范检查',
    description: '检查标题、字数和基本字段完整性。',
    scriptDescription: '检查标题、字数和基本字段完整性。',
    interpreter: 'python3.11',
    isEnabled: true,
    activeVersionID: 11,
    meta: { runtime: 'python3.11' },
    extensionData: { runtime: 'python3.11' },
    createdAt: '2026-03-15 10:30',
  },
  {
    scriptID: 2,
    scriptKey: 'sensitive-filter',
    scriptName: '敏感词过滤',
    description: '提交前自动扫描敏感词并标记结果。',
    scriptDescription: '提交前自动扫描敏感词并标记结果。',
    interpreter: 'python3.11',
    isEnabled: true,
    activeVersionID: 21,
    meta: { runtime: 'python3.11' },
    extensionData: { runtime: 'python3.11' },
    createdAt: '2026-03-20 09:10',
  },
  {
    scriptID: 3,
    scriptKey: 'ai-detector',
    scriptName: 'AI 文本提示检测',
    description: '辅助检测机器生成痕迹，输出风险分。',
    scriptDescription: '辅助检测机器生成痕迹，输出风险分。',
    interpreter: 'python3.11',
    isEnabled: false,
    activeVersionID: 31,
    meta: { runtime: 'python3.11' },
    extensionData: { runtime: 'python3.11' },
    createdAt: '2026-03-25 13:40',
  },
]

export const mockScriptVersionsByScript: Record<number, ScriptVersion[]> = {
  1: [
    {
      versionID: 11,
      scriptID: 1,
      versionNum: 1,
      versionName: 'v1.0.0',
      fileName: 'format_checker.py',
      relativePath: 'scripts/format-checker/v1/format_checker.py',
      checksum: 'sha256:format-checker-v1',
      createdBy: 1,
      isActive: true,
      createdAt: '2026-03-15 10:30',
    },
    {
      versionID: 12,
      scriptID: 1,
      versionNum: 2,
      versionName: 'v1.1.0',
      fileName: 'format_checker_v1_1.py',
      relativePath: 'scripts/format-checker/v2/format_checker_v1_1.py',
      checksum: 'sha256:format-checker-v2',
      createdBy: 1,
      isActive: false,
      createdAt: '2026-03-29 17:15',
    },
  ],
  2: [
    {
      versionID: 21,
      scriptID: 2,
      versionNum: 1,
      versionName: 'v1.0.0',
      fileName: 'sensitive_filter.py',
      relativePath: 'scripts/sensitive-filter/v1/sensitive_filter.py',
      checksum: 'sha256:sensitive-filter-v1',
      createdBy: 1,
      isActive: true,
      createdAt: '2026-03-20 09:10',
    },
  ],
  3: [
    {
      versionID: 31,
      scriptID: 3,
      versionNum: 1,
      versionName: 'v0.9.0',
      fileName: 'ai_detector.py',
      relativePath: 'scripts/ai-detector/v1/ai_detector.py',
      checksum: 'sha256:ai-detector-v1',
      createdBy: 1,
      isActive: true,
      createdAt: '2026-03-25 13:40',
    },
  ],
}

export const mockFlows: ScriptFlow[] = [
  {
    flowID: 1,
    flowKey: 'submission-precheck',
    flowName: '投稿预处理流程',
    description: '投稿后自动完成基础检查与敏感词扫描。',
    flowDescription: '投稿后自动完成基础检查与敏感词扫描。',
    isEnabled: true,
    meta: { trigger: 'work_created' },
    extensionData: { trigger: 'work_created' },
    createdAt: '2026-03-26 11:00',
  },
  {
    flowID: 2,
    flowKey: 'review-assistant',
    flowName: '复审辅助流程',
    description: '复审阶段调用检测脚本生成风险提醒。',
    flowDescription: '复审阶段调用检测脚本生成风险提醒。',
    isEnabled: false,
    meta: { trigger: 'review_assigned' },
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
      scriptVersionID: 11,
      isEnabled: true,
      failureStrategy: 'STOP',
      inputTemplate: { rejectOnError: true },
      timeoutMs: 5000,
      stepConfig: { rejectOnError: true },
    },
    {
      stepID: 102,
      flowID: 1,
      stepOrder: 2,
      stepName: '敏感词扫描',
      scriptID: 2,
      scriptVersionID: 21,
      isEnabled: true,
      failureStrategy: 'CONTINUE',
      inputTemplate: { blockLevel: 'medium' },
      timeoutMs: 5000,
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
      scriptVersionID: 31,
      isEnabled: true,
      failureStrategy: 'CONTINUE',
      inputTemplate: { threshold: 0.8 },
      timeoutMs: 7000,
      stepConfig: { threshold: 0.8 },
    },
  ],
}

export const mockFlowMountsByFlow: Record<number, FlowMount[]> = {
  1: [
    {
      mountID: 4,
      flowID: 1,
      scope: 'global',
      targetType: 'global',
      targetID: 0,
      eventKey: 'work_created',
      isEnabled: true,
      containerType: 'global',
      containerID: 0,
      createdAt: '2026-03-26 10:00',
    },
    {
      mountID: 1,
      flowID: 1,
      scope: 'track',
      targetType: 'track',
      targetID: 101,
      eventKey: 'work_created',
      isEnabled: true,
      containerType: 'track',
      containerID: 101,
      mountConfig: { phase: 'submission' },
      createdAt: '2026-03-26 11:30',
    },
    {
      mountID: 2,
      flowID: 1,
      scope: 'track',
      targetType: 'track',
      targetID: 102,
      eventKey: 'work_created',
      isEnabled: true,
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
      scope: 'contest',
      targetType: 'contest',
      targetID: 1,
      eventKey: 'review_assigned',
      isEnabled: true,
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

export const mockJudgeProfiles: JudgeProfile[] = [
  {
    judgeID: 301,
    judgeName: '吴穹',
    judgeEmail: 'wuqiong@ubik.com',
    isActive: true,
  },
  {
    judgeID: 302,
    judgeName: '许临',
    judgeEmail: 'xulin@ubik.com',
    isActive: true,
  },
  {
    judgeID: 303,
    judgeName: '裴澈',
    judgeEmail: 'peiche@ubik.com',
    isActive: true,
  },
]

export const mockReviewEvents: ReviewEvent[] = [
  {
    eventID: 701,
    trackID: 101,
    eventName: '硬核初评',
    workStatus: 'submission_success',
    startTime: '2026-04-08T09:00:00Z',
    endTime: '2026-04-18T23:59:59Z',
    judgeIDs: [301, 302],
  },
  {
    eventID: 702,
    trackID: 102,
    eventName: '太空歌剧初评',
    workStatus: 'submission_success',
    startTime: '2026-04-09T09:00:00Z',
    endTime: '2026-04-19T23:59:59Z',
    judgeIDs: [301, 303],
  },
  {
    eventID: 703,
    trackID: 201,
    eventName: '推理初评',
    workStatus: 'submission_success',
    startTime: '2026-04-10T09:00:00Z',
    endTime: '2026-04-20T23:59:59Z',
    judgeIDs: [302, 303],
  },
]

export const mockReviewResultsByWork: Record<number, ReviewResult[]> = {
  5001: [
    {
      resultID: 9001,
      workID: 5001,
      reviewEventID: 701,
      reviews: {
        finalScore: 88.5,
        reviewCount: 2,
        assignedJudgeCount: 2,
        comments: '结构完整，世界观清晰。',
        judgeScores: { '301': 90, '302': 87 },
        generatedAt: '2026-04-11T06:00:00Z',
      },
    },
  ],
  5002: [
    {
      resultID: 9002,
      workID: 5002,
      reviewEventID: 701,
      reviews: {
        finalScore: 92.0,
        reviewCount: 2,
        assignedJudgeCount: 2,
        comments: '人物立得住，节奏优秀。',
        judgeScores: { '301': 93, '302': 91 },
        generatedAt: '2026-04-11T06:00:00Z',
      },
    },
  ],
  5003: [
    {
      resultID: 9003,
      workID: 5003,
      reviewEventID: 702,
      reviews: {
        finalScore: 85.5,
        reviewCount: 2,
        assignedJudgeCount: 2,
        comments: '创意不错，细节仍可加强。',
        judgeScores: { '301': 86, '303': 85 },
        generatedAt: '2026-04-12T06:00:00Z',
      },
    },
  ],
}

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
      workStatus: '待审核',
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
      workStatus: '已通过',
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
      workStatus: '待审核',
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
      workStatus: '已驳回',
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
      workStatus: '待审核',
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
