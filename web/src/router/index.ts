import { createRouter, createWebHistory } from "vue-router";
export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      component: () => import("../features/topology/TopologyWorkspace.vue"),
    },
    {
      path: "/templates",
      component: () => import("../views/TemplatesView.vue"),
    },
    {
      path: "/automation",
      component: () => import("../views/AutomationView.vue"),
    },
  ],
});
