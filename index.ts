const os = require("os");
const net = require("net");
const cp = require("child_process");
const fs = require("fs");
const cluster = require("cluster");

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
    } catch (e) {
        console.log("Installing modules");
        cp.execSync("npm i", {stdio: [0, 1, 2]});
        console.log("Restarting");

        process.exit();
    }


    let args = process.argv.slice(2);
    var host = "localhost";
    if (args.length) {
        host = args[0];
    }

    storage.initSync({dir: "tmp/" + os.hostname()});
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

    function writeToJLink(lines, cb = null) {
        (function iter(i) {
            if (i >= lines.length) {
                if (cb) cb();
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


    function setCore(core: string) {
        if (settings.core != core) {
            settings.core = core;
            settings.device = core == "Cortex-M4" ? "NRF52832_XXAA" : "NRF51822_XXAC";
            jLinkProc.kill("SIGINT");
        }

        socket.emit("settings", settings);
    }


    (function runJLink() {
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
        jLinkProc.stdout.on("data", function (data: string) {
            // console.log("stdout:", data);
            if (data.indexOf("FAILED: Can not connect to J-Link via USB") >= 0) {
                setTimeout(function () {
                    jLinkProc.kill("SIGINT");
                }, 1000);
            }

            let m;
            if (m = data.match(/(Cortex-.*?) identified/)) setCore(m[1]);

            socket.emit("jlink.data", {output: "stdout", data: data})
        });
        jLinkProc.stderr.on("data", function (data) {
            console.log("stderr:", data);

            socket.emit("jlink.data", {output: "stderr", data: data})
        });
        jLinkProc.on("close", function (code) {
            socket.emit("jlink.close", {code: code, action: jLinkAction});
            console.log("JLINK closed");
            runJLink();
        });
    })();

    socket.on("connect", function () {
        socket.emit("node", {hostname: os.hostname()});
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
        var tmpFile = tmp.fileSync({postfix: ".hex"}).name;

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
        var tmpFile = tmp.fileSync({postfix: ".hex"}).name;
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

    function createRTTConnection(host: string, port: number) {
        (function init() {
            let conn = net.createConnection({host: host, port: port}, function () {
                console.log("Connected to " + host + ":" + port);
            });
            conn.on("error", () => {
            });
            conn.on("data", function (data) {
                process.stdout.write(data.toString());
                // console.log(Date.now() + ": " + data.toString().trim());
                socket.emit("rtt", data.toString());
            });
            conn.on("close", function () {
                setTimeout(init, 300);
            });
        })();
    }

    createRTTConnection("localhost", 19021);
}
