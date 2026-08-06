import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import { router } from "./router";
import { bootstrapTheme } from "./themeBootstrap";
import "./styles/index.css";

bootstrapTheme();
createApp(App).use(createPinia()).use(router).mount("#app");
