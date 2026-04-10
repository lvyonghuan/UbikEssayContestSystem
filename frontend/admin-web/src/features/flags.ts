export const featureFlags = {
  judgeModule: import.meta.env.VITE_ENABLE_JUDGE_MODULE === 'true',
  scriptModule: import.meta.env.VITE_ENABLE_SCRIPT_MODULE !== 'false',
}
