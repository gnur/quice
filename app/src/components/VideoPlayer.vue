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
import axios from 'axios'

export default {
  name: 'VideoPlayer',
  props: [ "user", "playlist" ],
    data: function () {
      return {
        video: {}
      }
    },
    methods: {
      startRecord: function () {
        console.log("starting recorder")
        var vm = this
        vm.timer = setInterval(this.updatePlayStatus, 2900)
      },
      updatePlayStatus: function () {
        var vm = this
        var player = document.getElementById("videoplayer");
        var pos = Math.floor(player.currentTime);
        var key = encodeURIComponent(this.video.key);
        axios.post("/api/updatecurrent/", {
            user: vm.user,
            key: vm.video.key,
            playlist: vm.playlist,
            position: pos,
          }).then((response) => {
          console.log("ok")
       }, (error) => {
         console.log(error)
       })
      },
      gotoNextVideo: function () {
        //set current as complete
        var vm = this;
        axios.post("/api/setcompleted/", {
            user: vm.user,
            key: vm.video.key,
            playlist: vm.playlist,
          }).then((response) => {
          console.log("ok")
        this.playVideo()
       }, (error) => {
         console.log(error)
       })
        console.log("completed")
      },
      stopRecord: function () {
        var vm = this
        console.log("stopping recorder")
        clearInterval(vm.timer)
      },
      playVideo: function () {
        console.log("starting next video")
        axios.get('/api/current/' + this.user + '/' + this.playlist + '/').then((response) => {
          var resp = response.data;
          this.video = resp;
          var player = document.getElementById("videoplayer");
          var source = document.getElementById('videosource');

          source.setAttribute('src', resp.url);
          player.load()
          player.currentTime = resp.pos;
          player.play()
       }, (error) => {
         console.log(error)
       })
      }
    },
    mounted: function () {
      var player = document.getElementById("videoplayer");
      var source = document.createElement('source');
      source.setAttribute("id", "videosource");
      player.appendChild(source);
      this.playVideo();
    }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
h1, h2 {
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
