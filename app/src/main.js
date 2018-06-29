// The Vue build version to load with the `import` command
// (runtime-only or standalone) has been set in webpack.base.conf with an alias.
import './../node_modules/bulma/css/bulma.css';
import Vue from 'vue'
import PlaylistSelect from './components/PlaylistSelect'
import UserSelect from './components/UserSelect'
import VideoPlayer from './components/VideoPlayer'
import App from './App'
import VueRouter from 'vue-router'


Vue.use(VueRouter)
const routes = [
  { path: '/', component: UserSelect },
  { path: '/user/:user', name: "PlaylistSelect", component: PlaylistSelect, props: true },
  { path: '/user/:user/:playlist', name: "VideoPlayer", component: VideoPlayer, props: true }
]

// Create the router instance and pass the `routes` option
// You can pass in additional options here, but let's
// keep it simple for now.
const router = new VueRouter({
  routes, // short for routes: routes
})
new Vue({
  el: '#app',
  template: '<App/>',
  components: { App },
  router
})
