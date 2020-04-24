let MAXOP = 10000;
let statistic = {}, // name -> id
    tags = [],      // { c: count, r: raw html, s: sub node count, l: max length of branch, p: placement { x, y, r, s } } (sort by count)
    datapacks = [], // { n: name, u: uid }
    relations = []; // []{ c: relation count, w: summary of weight} relation table
let sum = 0, // amount of tags
    root = null, tags_old = null; // root node
function getTagsInDatapack(d) {
    let elements = []
    Array.prototype.forEach.call(d.children[1].children, function (t) {
        if (t.classList.contains("tag-2") || t.classList.contains("tag-3")) {
            elements.push(t);
        }
    });
    return elements;
}

// Read
Array.prototype.forEach.call(document.querySelectorAll('.datapack-short'), function (d) {
    elements = getTagsInDatapack(d)
    for (let i = 0; i < elements.length; i++) {
        if (statistic.hasOwnProperty(elements[i].textContent)) {
            tags[statistic[elements[i].textContent]].c++;
        }
        else {
            statistic[elements[i].textContent] = sum;
            tags.push({
                c: 1,
                r: elements[i],
                s: 0,
                p: {
                    x: 0,   // position
                    y: 0,
                    r: 0,   // radius
                    s: 0    // sigma
                }
            })
            sum++;
        }
    }
    datapacks.push({
        n: d.children[0].textContent,
        u: d.children[0].id,
        t: d.children[1]
    });
}); // read data
tags.sort(function(a,b) {
    return b.c-a.c;
}) // set rank
for (let i = 0; i < sum; i++) {
    statistic[tags[i].textContent] = i;
} // update index of statistics, inverted

// Calculate
for (let i = 0; i < sum; i++) {
    relations[i] = [];
    for (let j = 0; j < sum; j++)
        relations[i][j] = {
            c: 0,
            w: 0
        };
} // init relation table
Array.prototype.forEach.call(document.querySelectorAll('.datapack-short'), function (d) {
    elements = getTagsInDatapack(d)
    for (let i = 0; i < elements.length; i++) {
        let vi = statistic[elements[i].textContent];
        for (let j = 0; j < elements.length; j++) {
            if (elements[j] !== elements[i]) {
                let vj = statistic[elements[j].textContent];
                if (vi < vj)
                    relations[vi][vj].c++;
            }
        }
    }
}); // build relation table in partial order
for (let i = sum - 1; i >= 0; i--) {
    let nw = tags[i].c;
    for (let j = i + 1; j < sum; j++) {
        if (relations[i][j].c > 0) {
            relations[i][j].w = relations[j][j].w - tags[j].c;
            nw += relations[i][j].w;
        }
    }
    for (let j = i + 1; j < sum; j++) {
        if (relations[i][j].c > 0) {
            relations[j][i].w = nw;
        }
    }
    relations[i][i].w = nw;
} // set weight

// Canvas
Array.prototype.forEach.call(document.querySelectorAll('.tag-unique'), function (e) {
    root = e.parentNode;
    tags_old = e;
}); // remove all
let canvas = document.createElement("canvas")
canvas.width = document.documentElement.clientWidth;
canvas.height = document.documentElement.clientHeight;
canvas.style.zIndex = "1000";
canvas.id = "tags";
document.body.appendChild(canvas); // add tags panel
datapacks = document.createElement("div")
datapacks.id = "datapacks"; // add datapacks panel
document.body.appendChild(datapacks);

// Draw
function Draw() {
    root.removeChild(tags_old);
    let ctx = document.getElementById("tags").getContext("2d"),
        cx = document.documentElement.clientWidth / 2, cy = document.documentElement.clientHeight / 2;
    let scale = 1, root_sigma = Math.PI / 2,
        x_min = 0, y_min = 0, x_max = 0, y_max = 0, // canvas params
        x_mass = 0, y_mass = 0; // mass of the whole
    function place(i) {
        let n = 0, _j = 0, _x_mass = 0, _y_mass = 0, r = Math.sqrt(tags[i].c);
        for (let j = 0; j < i; j++) {
            if (relations[j][i].c > 0) {
                n++;
                _x_mass += tags[j].p.x;
                _y_mass += tags[j].p.y;
                _j = j;
            }
        }
        if (n === 1) { // expanding node, outside net
            let sigma = tags[_j].p.s, sw = - tags[_j].c, lw = 0;
            for (let j = 0; j <= _j; j++) sw += relations[_j][j].w; // sum of weight
            for (let j = _j + 1; j < i; j++) lw += relations[j][_j].w; // sum of weight of children of _j before i
            sigma += (sw + tags[_j].c - relations[_j][_j].w) / sw / 2 * Math.PI - Math.PI; // relative origin angle
            let _sigma = (lw + relations[_j][i].w / 2) / sw * Math.PI * 2 + sigma; // edge angle
            if (i === 1) {
                _sigma = root_sigma;
            }
            let px = tags[_j].p.x + Math.cos(_sigma) * (r + Math.sqrt(tags[_j].c)),
                py = tags[_j].p.y + Math.sin(_sigma) * (r + Math.sqrt(tags[_j].c));
            let __s = confine(i, px, py, r, _j);
            if (__s !== null) _sigma = __s;
            tags[i].p.x = tags[_j].p.x + Math.cos(_sigma) * (r + Math.sqrt(tags[_j].c));
            tags[i].p.y = tags[_j].p.y + Math.sin(_sigma) * (r + Math.sqrt(tags[_j].c));
        } else if (n > 1) { // binding node, inside net
            tags[i].p.x = _x_mass / n;
            tags[i].p.y = _y_mass / n;
        }
        tags[i].p.r = r;
        if (i === 0) {
            tags[i].p.s = root_sigma;
        } else {
            tags[i].p.s = Math.atan2(tags[i].p.y - _y_mass / n, tags[i].p.x - _x_mass / n);
        }
        if (i > 0) fix(i, _j);

        x_mass += tags[i].p.x;
        y_mass += tags[i].p.y;

        x_min = Math.min(x_min, tags[i].p.x - r);
        y_min = Math.min(y_min, tags[i].p.y - r);
        x_max = Math.max(x_max, tags[i].p.x + r);
        y_max = Math.max(y_max, tags[i].p.y + r);
        /*
        let t = tags[i],
            r = Math.sqrt(t.c),                             // radius of the tag
            sw = - t.c,                                     // total weight of sub nodes and parent nodes
            lw = 0;                                         // last weight
        for (let j = 0; j <= i; j++) sw += relations[i][j].w;
        sigma = sigma + (sw + t.c - relations[i][i].w) / sw / 2 * Math.PI - Math.PI; // relative origin angle
        for (let j = i + 1; j < sum; j++) {
            if (relations[i][j].c > 0) {
                let _r = Math.sqrt(tags[j].c),
                    _sigma = (lw + relations[i][j].w / 2) / sw * Math.PI * 2 + sigma,  // edge angle
                    _x = x + Math.cos(_sigma) * (r + _r) * 2,
                    _y = y + Math.sin(_sigma) * (r + _r) * 2;
                lw += relations[i][j].w;
                tags[j].p.p.push(i);
                tags[i].p.x = x; tags[i].p.y = y; tags[i].p.r = r; tags[i].p.s = sigma;
            }
        }*/
    }
    function confine(i, px, py, r, j) { // add a drag if exceeded max margin
        let _pr = Math.max(x_max, y_max), _nr = Math.min(x_min, y_min), dr = Math.max(Math.abs(_nr), _pr);
        let theta_1 = Math.atan2(tags[j].p.y - (x_min + x_max) / 2, tags[j].p.x - (y_min + y_max) / 2), // vector to center of graph
            theta_2 = Math.atan2(py - tags[j].p.y, px - tags[j].p.x);
        if (px * px + py * py > dr * dr) theta_1 = Math.atan2(tags[j].p.y - (y_mass / i) / 2, tags[j].p.x - (x_mass / i) / 2) // exceed
        if (Math.sin(theta_2 - theta_1) > 0) return theta_1 + Math.PI / 2;
        else return theta_1 - Math.PI / 2;
    }
    function fix(i, j) { // fix when being overlapped, find a gap to place it again and keep balance of the graph.
        let _x_mass = 0, _y_mass = 0, r = tags[i].p.r, n = 0;
        for (let j = 0; j < i; j++) { // collide detection
            if ((tags[j].p.x - tags[i].p.x) * (tags[j].p.x - tags[i].p.x) + (tags[j].p.y - tags[i].p.y) * (tags[j].p.y - tags[i].p.y) < (tags[j].p.r + r) * (tags[j].p.r + r)) {
                _x_mass += tags[j].p.x;
                _y_mass += tags[j].p.y;
                n++;
            }
        }
        if (n === 0) {
            return
        }
        if (_x_mass === 0 && _y_mass === 0) _x_mass = 1;
        let v = vector_add([_x_mass / n - x_mass / i, _y_mass / n - y_mass / i], [(x_min + x_max) / 2 - x_mass / i, (y_min + y_max) / 2 - y_mass / i]),
            _diff_x = (v[0] === 0) ? 1e-16 : 0, _diff_y = (v[1] === 0) ? 1e-16 : 0;
        let k = - (v[0] + _diff_x) / (v[1] + _diff_y), b = _y_mass / n - k * _x_mass / n,
            c = (tags[i].p.x / k + tags[i].p.y - b) / (k + 1 / k), clipped = [];
        function vector_add(a, b) {
            return [(a[0] + b[0]) / 2, (a[1] + b[1]) / 2];
        }
        function find_gap() {
            for (let j = 0; j < i; j++) {
                let d = Math.abs(k * tags[j].p.x - tags[j].p.y + b) / Math.sqrt(k * k + 1);
                if (d < tags[j].p.r + r) { // if connect to the line
                    let l = Math.sqrt((tags[j].p.r + r) * (tags[j].p.r + r) - d * d) / Math.sqrt(k * k + 1),
                        cl = (tags[j].p.x / k + tags[j].p.y - b) / (k + 1 / k);
                    clipped.push({l: cl-l, r: cl+l});
                }
            }
            clipped.sort(function(a,b) {
                return a.l-b.l;
            });
            let ans_l = c, ans_r = c;
            if (clipped.length > 0) {
                ans_l = clipped[0].l;
                ans_r = clipped[0].r;
            }
            for (let j = 1; j < clipped.length; j++) {
                if (clipped[j].l >= ans_r) { // find gap
                    if (ans_r > c) break; // get answer
                    ans_l = clipped[j].l;
                    if (ans_l > c) break; // get answer
                    ans_r = clipped[j].r;
                } else { // connecting clip
                    if (clipped[j].r > ans_r) ans_r = clipped[j].r;
                }
            }
            if (ans_l < ans_r) { // dilemma
                if (c - ans_l < ans_r - c) tags[i].p.x = ans_l;
                else tags[i].p.x = ans_r;
            } else tags[i].p.x = c;
            tags[i].p.y = k * tags[i].p.x + b;
        }
        find_gap();
    }
    let progress = 0;
    function buildAnimation() {
        place(progress);
        let t = tags[progress];
        ctx.beginPath();
        ctx.arc(t.p.x * 2 + cx, t.p.y * 2 + cy, t.p.r * 2, 0 ,2*Math.PI);
        ctx.stroke();
        if (progress < sum - 1) {
            progress += 1;
            window.requestAnimationFrame(buildAnimation);
        }
    }
    window.requestAnimationFrame(buildAnimation);
}

function main() {

    window.requestAnimationFrame(main);
}
//window.requestAnimationFrame(main);
Draw();