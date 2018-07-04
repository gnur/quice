<template>
<section>
<nav class="level">
  <p class="level-item has-text-centered">
      <router-link :to="{ name: 'UserSelect' }" class="link is-info">
          <p class="title">users</p>
        </router-link>
  </p>
  <p class="level-item has-text-centered">
      <router-link :to="{ name: 'PlaylistSelect', params: { user: user }}" class="link is-info">
          <p class="title">playlists</p>
        </router-link>
  </p>
</nav>
<div class="tile is-ancestor">
  <div class="tile is-parent">
    <div class="tile is-3 is-child box">
      <progress class="progress is-info" :value="currentVideo" :max="totalVideos">{{ currentVideo }}/{{ totalVideos }}</progress><br>
      <p class="title">{{ currentVideo }}/{{ totalVideos }}</p>
      {{ video.key | keyToNice }}
    </div>
    <div class="tile is-child box">
        <video id="videoplayer"
            controls="true"
            type="video/mp4"
            v-on:ended="gotoNextVideo"
            v-on:play="startRecord"
            v-on:pause="stopRecord"
            class="is-8">
        </video>
    </div>
  </div>
</div>
  </section>
</template>

<script>
import axios from "axios";

export default {
  name: "VideoPlayer",
  props: ["user", "playlist"],
  data: function() {
    return {
      video: {},
      currentVideo: 2,
      totalVideos: 4,
      all: [],
    };
  },
  filters: {
    keyToNice(value) {
      if (!value) return '';
      return value.split("/")[0].replace("_", " ");
    }
  },
  methods: {
    stopRecord: function() {
      var vm = this;
      clearInterval(vm.timer);
      vm.updatePlayStatus();
    },
    startRecord: function() {
      var vm = this;
      vm.timer = setInterval(this.updatePlayStatus, 5000);
    },
    updatePlayStatus: function() {
      var vm = this;
      var player = document.getElementById("videoplayer");
      var pos = Math.floor(player.currentTime) - 10;
      var key = encodeURIComponent(this.video.key);
      axios
        .post("/api/updatecurrent/", {
          user: vm.user,
          key: vm.video.key,
          playlist: vm.playlist,
          position: pos
        })
        .then(
          response => {},
          error => {
            console.log(error);
          }
        );
    },
    gotoNextVideo: function() {
      var vm = this;
      axios
        .post("/api/setcompleted/", {
          user: vm.user,
          key: vm.video.key,
          playlist: vm.playlist
        })
        .then(
          response => {
            var player = document.getElementById("videoplayer");
            this.playVideo(true);
          },
          error => {
            console.log(error);
          }
        );
    },
    playVideo: function(autostart) {
      axios.get("/api/current/" + this.user + "/" + this.playlist + "/").then(
        response => {
          var resp = response.data;
          this.video = resp;
          this.currentVideo = resp.all.indexOf(resp.key);
          this.totalVideos = resp.all.length;
          this.all = resp.all;
          var player = document.getElementById("videoplayer");
          var source = document.getElementById("videosource");

          source.setAttribute("src", resp.url);
          player.load();
          var handler = function() {
            player.currentTime = resp.pos;
            if (autostart) {
              player.play();
            }
            player.removeEventListener("loadeddata", handler);
          };
          player.addEventListener("loadeddata", handler);
        },
        error => {
          console.log(error);
        }
      );
    }
  },
  mounted: function() {
    var player = document.getElementById("videoplayer");
    var source = document.createElement("source");
    source.setAttribute("id", "videosource");
    player.appendChild(source);
    this.playVideo(false);
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
</style>
