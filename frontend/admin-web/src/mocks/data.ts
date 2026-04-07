import type { Contest, Track } from '@/types/api'

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
