<template>
  <section class="not-found">
    <h1>404</h1>
    <p>页面不存在或链接已失效。</p>
    <router-link to="/dashboard">返回比赛看板</router-link>
  </section>
</template>

<style scoped>
.not-found {
  min-height: 100vh;
  display: grid;
  place-items: center;
  gap: 8px;
  text-align: center;
}

.not-found h1 {
  margin: 0;
  font-size: 56px;
}
</style>
