<template>
  <section>
    <nav class="navbar is-transparant" role="navigation">
      <div class="navbar-menu is-active">
        <div class="navbar-start">
          <router-link :to="{ name: 'UserSelect' }" class="navbar-item">users</router-link>
          <router-link :to="{ name: 'PlaylistSelect' }" class="navbar-item">playlists</router-link>
        </div>
      </div>
    </nav>

    <div class="tile is-ancestor">
      <div class="tile is-parent is-4">
        <div class="tile is-child box has-background-grey-lighter has-text-grey-dark" v-if="loaded">
          <progress
            class="progress"
            :value="currentVideo"
            :max="totalVideos"
          >{{ currentVideo }}/{{ totalVideos }}</progress>
          <router-link :to="{ name: 'PlaylistEditor' }">
            <p class="title">{{ currentVideo+1 }}/{{ totalVideos }}</p>
          </router-link>
          <br>
          <p></p>
          <p class="title is-5">Now playing:</p>
          <p class="subtitle is-6" :title="video.key">{{ video.filename | keyToNice }}</p>
          <p></p>
          <h6 class="title is-5">Next up:</h6>
          <a
            v-on:click="gotoNextVideo"
          >{{ allVideos[sortedKeys[currentVideo + 1]].filename | keyToNice }}</a>
        </div>
        <div class="tile is-child box has-background-grey-lighter has-text-grey-dark" v-else>
          <p class="title is-2">Loading...</p>
        </div>
      </div>

      <div class="tile is-parent">
        <div class="tile is-child box has-background-grey-lighter has-text-grey-dark">
          <video
            id="videoplayer"
            controls="true"
            type="video/mp4"
            v-on:ended="gotoNextVideo"
            v-on:play="startRecord"
            v-on:pause="stopRecord"
            class="is-7"
          ></video>
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
      loaded: false,
      currentVideo: 2,
      totalVideos: 4,
      completed: false,
      sortedKeys: [],
      allVideos: {}
    };
  },
  filters: {
    keyToNice(value) {
      if (!value) return "";
      console.log(value);
      value = value.replace("^[^/]+", "");
      return value.replace(/(_|\/|\.)/g, " ");
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
      vm.loaded = false;
      axios
        .post("/api/setcompleted/", {
          user: vm.user,
          key: vm.video.key,
          playlist: vm.playlist
        })
        .then(
          response => {
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
          var vm = this;
          this.totalVideos = resp.sortedKeys.length;
          this.sortedKeys = resp.sortedKeys;
          this.allVideos = resp.videos;
          if (resp.completed) {
            this.currentVideo = this.totalVideos - 1;
            vm.loaded = true;
            return;
          }
          this.video = this.allVideos[resp.key];

          this.currentVideo = resp.sortedKeys.indexOf(resp.key);
          var player = document.getElementById("videoplayer");
          var source = document.getElementById("videosource");

          source.setAttribute("src", resp.url);
          player.load();
          var handler = function() {
            vm.loaded = true;
            player.currentTime = resp.pos;
            document.title = vm.video.key;
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
