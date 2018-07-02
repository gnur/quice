<template>
  <div class="tile is-ancestor is-10">

        <video id="videoplayer"
            controls="true"
            type="video/mp4"
            v-on:ended="gotoNextVideo"
            v-on:play="startRecord"
            v-on:pause="stopRecord"
            class="is-8">
        </video>

  </div>
</template>

<script>
import axios from "axios";

export default {
  name: "VideoPlayer",
  props: ["user", "playlist"],
  data: function() {
    return {
      video: {}
    };
  },
  methods: {
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
            if (document.fullscreenEnabled) {
              document.exitFullscreen();
            }
            this.playVideo();
          },
          error => {
            console.log(error);
          }
        );
    },
    stopRecord: function() {
      var vm = this;
      clearInterval(vm.timer);
    },
    playVideo: function() {
      axios.get("/api/current/" + this.user + "/" + this.playlist + "/").then(
        response => {
          var resp = response.data;
          this.video = resp;
          var player = document.getElementById("videoplayer");
          var source = document.getElementById("videosource");

          source.setAttribute("src", resp.url);
          player.load();
          player.addEventListener("loadeddata", function() {
            player.currentTime = resp.pos;
            player.play();
            player.removeEventListener("loadeddata");
          });
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
    this.playVideo();
  }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
h1,
h2 {
  font-weight: normal;
}
ul {
  list-style-type: none;
  padding: 0;
}
li {
  display: inline-block;
  margin: 0 10px;
}
a {
  color: #42b983;
}
</style>
