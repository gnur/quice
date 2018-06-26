var s3player = new Vue({
  el: '#app',
  data: {
    videos: [
      "https://s3.erwin.land/golpje/ridiculousness/season%2006/ridiculousness.0630-yestv.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=erwin%2F20180626%2Fus-west-1%2Fs3%2Faws4_request&X-Amz-Date=20180626T074924Z&X-Amz-Expires=432000&X-Amz-SignedHeaders=host&X-Amz-Signature=9bc34dc17111ed773a8ff2e1440eb4f1a23bf612168f13825768535230e32d43",
      "https://s3.erwin.land/golpje/daily%20show/2018-06/the.daily.show.2018.06.21.mike.shinoda.extended.web.x264-tbs%5Bettv%5D.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=erwin%2F20180626%2Fus-west-1%2Fs3%2Faws4_request&X-Amz-Date=20180626T094420Z&X-Amz-Expires=432000&X-Amz-SignedHeaders=host&X-Amz-Signature=49e63a7050f267b049ee0740d79c0745a83c5e141140f2dde49d780b6fc5ab13",
    ],
    position: 0,
  },
  created: function () {
    var vm = this;
    var player = document.getElementById("videoplayer");
    var source = document.createElement('source');
    source.setAttribute('src', vm.videos[vm.position]);
    source.setAttribute("id", "videosource");
    player.appendChild(source);
    player.load()
  },

  methods: {
    startRecord: function () {
      console.log("starting recorder")
      var vm = this
      vm.timer = setInterval(this.updatePlayStatus, 2900)
    },
    stopRecord: function () {
      var vm = this
      console.log("stopping recorder")
      clearInterval(vm.timer)
    },
    gotoNextVideo: function () {
      var vm = this
      console.log("starting next video")
      var player = document.getElementById("videoplayer");
      var source = document.getElementById('videosource');
      vm.position++;
      source.setAttribute('src', vm.videos[vm.position]);
      player.load()
      player.play()
    },
    getGif: function () {
      var vm = this
      axios.get('/getGifId', {})
        .then(function (response) {
          vm.gifId = response.data.id;
          vm.gifUrl = "https://giphy.com/embed/" + vm.gifId;
        })
        .catch(function (error) {
          console.log(error)
        })
    },
    updatePlayStatus: function () {
      var vm = this
      var player = document.getElementById("videoplayer");
      console.log(player.currentTime);
    }
  }
})
