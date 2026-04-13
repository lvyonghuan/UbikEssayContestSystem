export const featureFlags = {
  judgeModule: import.meta.env.VITE_ENABLE_JUDGE_MODULE !== 'false',
  scriptModule: import.meta.env.VITE_ENABLE_SCRIPT_MODULE !== 'false',
}
