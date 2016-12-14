var os = require("os");
var net = require("net");
var cp = require("child_process");
var fs = require("fs");
var cluster = require("cluster");
if (cluster.isMaster) {
    cluster.fork();
    cluster.on("exit", function (worker, code, signal) {
        cluster.fork();
    });
}
if (cluster.isWorker) {
    try {
        var io = require("socket.io-client");
        var tmp = require("tmp");
        var storage = require("node-persist");
        var moment = require("moment");
        var colors = require("colors");
    }
    catch (e) {
        console.log("Installing modules");
        cp.execSync("npm i", { stdio: [0, 1, 2] });
        console.log("Restarting");
        process.exit();
    }
    var args = process.argv.slice(2);
    var host = "localhost";
    if (args.length) {
        host = args[0];
    }
    if (args.indexOf("gdb") == -1) {
        runJLink();
    }
    storage.initSync({ dir: "tmp/" + os.hostname() });
    var settings = storage.getItemSync("settings");
    if (!settings) {
        settings = {
            device: "NRF52832_XXAA",
            core: "Cortex-M4",
            debug: true,
        };
    }
    console.log(settings);
    var socket = io.connect("http://" + host + ":3000");
    var jLinkProc;
    var jLinkAction = null;
    function writeToJLink(lines, cb) {
        if (cb === void 0) { cb = null; }
        (function iter(i) {
            if (i >= lines.length) {
                if (cb)
                    cb();
                return;
            }
            console.log("EXEC: ", lines[i]);
            jLinkProc.stdin.write(lines[i] + "\n", "utf8", function () {
                setTimeout(function () {
                    iter(i + 1);
                }, 300);
            });
        })(0);
    }
    function setCore(core) {
        if (settings.core != core) {
            settings.core = core;
            settings.device = core == "Cortex-M4" ? "NRF52832_XXAA" : "NRF51822_XXAC";
            jLinkProc.kill("SIGINT");
        }
        socket.emit("settings", settings);
    }
    function runJLink() {
        jLinkProc = cp.spawn("JLinkExe", ["-device", settings.device, "-speed", "4000", "-if", "SWD", "-autoconnect", "1"]);
        jLinkProc.stdout.setEncoding("utf8");
        jLinkProc.stderr.setEncoding("utf8");
        jLinkProc.on("error", function (err) {
            jLinkProc.kill("SIGINT");
            runJLink();
        });
        jLinkProc.stdin.on("error", function () {
            console.log("jLinkProc.stdin.on.error");
            console.log(arguments);
        });
        jLinkProc.stdout.on("data", function (data) {
            // console.log("stdout:", data);
            if (data.indexOf("FAILED: Can not connect to J-Link via USB") >= 0) {
                setTimeout(function () {
                    jLinkProc.kill("SIGINT");
                }, 1000);
            }
            var m;
            if (m = data.match(/(Cortex-.*?) identified/))
                setCore(m[1]);
            socket.emit("jlink.data", { output: "stdout", data: data });
        });
        jLinkProc.stderr.on("data", function (data) {
            console.log("stderr:", data);
            socket.emit("jlink.data", { output: "stderr", data: data });
        });
        jLinkProc.on("close", function (code) {
            socket.emit("jlink.close", { code: code, action: jLinkAction });
            console.log("JLINK closed");
            runJLink();
        });
    }
    socket.on("connect", function () {
        socket.emit("node", { hostname: os.hostname() });
        socket.emit("settings", settings);
    });
    socket.on("reset", function () {
        writeToJLink([
            "r",
            "g",
            "exit",
        ]);
    });
    socket.on("halt", function () {
        writeToJLink([
            "r",
        ]);
    });
    socket.on("erase", function () {
        writeToJLink([
            "w4 4001e504 2",
            "w4 4001e50c 1",
            "sleep 100",
            "w4 4001e504 1",
            "r",
            "exit",
        ]);
    });
    socket.on("flash", function (d) {
        var tmpFile = tmp.fileSync({ postfix: ".hex" }).name;
        settings.app = d.app;
        settings.board = d.board;
        settings.cflags = d.cflags;
        settings.debug = d.debug;
        console.log(d);
        storage.setItemSync("settings", settings);
        fs.writeFileSync(tmpFile, d.hex);
        writeToJLink([
            "r",
            "loadfile " + tmpFile,
            "r",
            "g",
            "exit",
        ], function () {
            fs.unlinkSync(tmpFile);
        });
    });
    socket.on("flashSoftDevice", function (d) {
        var tmpFile = tmp.fileSync({ postfix: ".hex" }).name;
        fs.writeFileSync(tmpFile, d.hex);
        writeToJLink([
            "w4 4001e504 2",
            "w4 4001e50c 1",
            "sleep 100",
            "r",
            "w4 4001e504 1",
            "loadfile " + tmpFile,
            "r",
            "g",
            "exit",
        ], function () {
            fs.unlinkSync(tmpFile);
        });
    });
    function ralign(x, l) {
        var r = "";
        for (var i = 0; i < l - x.length; i++)
            r += " ";
        r += x;
        return r;
    }
    function lalign(x, l) {
        var r = x;
        for (var i = 0; i < l - x.length; i++)
            r += " ";
        return r;
    }
    var maxFncLen_1 = 20;
    var maxModuleLen_1 = 4;
    function createRTTConnection(host, port) {
        var buffer = "";
        var currentTime = "";
        (function init() {
            var conn = net.createConnection({ host: host, port: port }, function () {
                console.log("Connected to " + host + ":" + port);
            });
            conn.on("error", function () {
            });
            conn.on("data", function (data) {
                if (buffer == "")
                    currentTime = moment().format("DD/MM/YY HH:mm:ss.SSS");
                buffer += data.toString();
                while (true) {
                    var pos = buffer.indexOf("\n");
                    if (pos == -1)
                        break;
                    var line = buffer.substr(0, pos);
                    process.stdout.write(colors.gray(currentTime + ": "));
                    // var rttEntry = {};
                    var info = line.match(/^(.*?):(\d+) <(.*?)> \(IRQ:(\d+?)\) @ (\d+): /);
                    if (info) {
                        var output = "";
                        var desc = info[1].match(/^(?:\[(\d)(?:,(.*?))?])?(.*)/);
                        var code = parseInt(desc[1]);
                        var module_1 = desc[2] ? desc[2] : "";
                        var file = desc[3];
                        var fncDesc = file + ":" + info[2] + " <" + info[3] + ">";
                        maxFncLen_1 = Math.max(fncDesc.length, maxFncLen_1);
                        maxModuleLen_1 = Math.max(module_1.length, maxModuleLen_1);
                        process.stdout.write("IRQ:" + ralign(info[4], 2) + " ");
                        output += ralign(module_1, maxModuleLen_1) + " ";
                        output += lalign(fncDesc, maxFncLen_1) + " ";
                        output += line.substr(info[0].length);
                        var color = void 0;
                        switch (code) {
                            case 9:
                                color = colors.green;
                                break;
                            case 2:
                                color = colors.red;
                                break;
                            case 1:
                                color = colors.yellow;
                                break;
                            case 0:
                                color = colors.white;
                                break;
                            default:
                                color = colors.white;
                        }
                        process.stdout.write(color(output));
                    }
                    else {
                        process.stdout.write(line);
                    }
                    process.stdout.write("\n");
                    buffer = buffer.substr(pos + 1);
                }
                socket.emit("rtt", data.toString());
            });
            conn.on("close", function () {
                setTimeout(init, 300);
            });
        })();
    }
    createRTTConnection("localhost", 19021);
}
