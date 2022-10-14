package main

process.env.PM2_USAGE = 'CLI';

import (
  cst "github.com/rprtr258/pm/internal" // constants

  // commander    commander
  // chalk        chalk
  // forEachLimit async/forEachLimit

  // debug        debug')('pm2:cli
  // PM2          ../API.js
  // pkg          ../../package.json
  // tabtab       ../completion.js
  // Common       ../Common.js
  // PM2ioHandler ../API/pm2-plus/PM2IO
)

Common.determineSilentCLI();
Common.printVersion();

var pm2 = new PM2();

PM2ioHandler.usePM2Client(pm2)

// function checkCompletion(){
//   return tabtab.complete('pm2', function(err, data) {
//     if(err || !data) return;
//     if(/^--\w?/.test(data.last)) return tabtab.log(commander.options.map(function (data) {
//       return data.long;
//     }), data);
//     if(/^-\w?/.test(data.last)) return tabtab.log(commander.options.map(function (data) {
//       return data.short;
//     }), data);
//     // array containing commands after which process name should be listed
//     var cmdProcess = ['stop', 'restart', 'scale', 'reload', 'delete', 'reset', 'pull', 'forward', 'backward', 'logs', 'describe', 'desc', 'show'];

//     if (cmdProcess.indexOf(data.prev) > -1) {
//       pm2.list(function(err, list){
//         tabtab.log(list.map(function(el){ return el.name }), data);
//         pm2.disconnect();
//       });
//     }
//     else if (data.prev == 'pm2') {
//       tabtab.log(commander.commands.map(function (data) {
//         return data._name;
//       }), data);
//       pm2.disconnect();
//     }
//     else
//       pm2.disconnect();
//   });
// };

if (_arr.indexOf('--no-daemon') > -1) {
  //
  // Start daemon if it does not exist
  //
  // Function checks if --no-daemon option is present,
  // and starts daemon in the same process if it does not exist
  //
  console.log('pm2 launched in no-daemon mode (you can add DEBUG="*" env variable to get more messages)');

  var pm2NoDaeamon = new PM2({
    daemon_mode : false
  });

  pm2NoDaeamon.connect(function() {
    pm2 = pm2NoDaeamon;
    beginCommandProcessing();
  });

}
else if (_arr.indexOf('startup') > -1 || _arr.indexOf('unstartup') > -1) {
  setTimeout(function() {
    commander.parse(process.argv);
  }, 100);
}
else {
  // HERE we instanciate the Client object
  pm2.connect(function() {
    debug('Now connected to daemon');
    if (process.argv.slice(2)[0] === 'completion') {
      checkCompletion();
      //Close client if completion related installation
      var third = process.argv.slice(3)[0];
      if ( third == null || third === 'install' || third === 'uninstall')
        pm2.disconnect();
    }
    else {
      beginCommandProcessing();
    }
  });
}
