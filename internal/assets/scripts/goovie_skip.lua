-- Goovie MPV Integration Script
-- Provides:
-- 1. Universal Netflix-like Skip Intro / Outro with OSD prompt and auto-skip.
-- 2. Multi-tier metadata detection (AniSkip / Scraper opts, MKV chapter parsing, unnamed chapter heuristics).
-- 3. Smart manual fallback jump (+85s) with instant undo ('U' / Backspace).
-- 4. Live buffering HUD during cache underruns.

local mp = require 'mp'
local opt = require 'mp.options'

local options = {
    op_start = -1,
    op_end = -1,
    ed_start = -1,
    ed_end = -1,
    skip_mode = "prompt", -- "prompt", "auto", "off"
    pos_file = "",
}
opt.read_options(options, "goovie")

local last_jump_pos = nil
local last_osd_show = 0
local last_save_time = 0

-- Persist current playback position to pos_file for resume
local function save_position()
    if not options.pos_file or options.pos_file == "" then
        return
    end
    local pos = mp.get_property_number("time-pos")
    local dur = mp.get_property_number("duration")
    if not pos or pos <= 0 then
        return
    end
    local f = io.open(options.pos_file, "w")
    if f then
        f:write(string.format('{"timePos": %.2f, "duration": %.2f}\n', pos, dur or 0))
        f:close()
    end
end

-- Scan chapters for opening/intro and ending/credits
local function scan_chapters()
    local chapters = mp.get_property_native("chapter-list", {})
    if not chapters or #chapters == 0 then
        return
    end

    for i, chap in ipairs(chapters) do
        local title = string.lower(chap.title or "")
        local start_t = chap.time or 0
        local next_chap = chapters[i + 1]
        local end_t = next_chap and next_chap.time or (start_t + 90)

        -- 1. Explicit chapter names
        if string.find(title, "intro") or string.find(title, "opening") or title == "op" or string.find(title, "theme") then
            if options.op_start < 0 then
                options.op_start = start_t
                options.op_end = end_t
            end
        elseif string.find(title, "outro") or string.find(title, "ending") or title == "ed" or string.find(title, "credit") then
            if options.ed_start < 0 then
                options.ed_start = start_t
                options.ed_end = end_t
            end
        end

        -- 2. Unnamed chapter heuristic: Chapter 2 in first 5 minutes lasting 40s - 100s
        if options.op_start < 0 and i == 2 and start_t >= 20 and start_t <= 300 then
            local dur = end_t - start_t
            if dur >= 40 and dur <= 100 then
                options.op_start = start_t
                options.op_end = end_t
            end
        end
    end
end

-- Key binding: Skip [S]
local function do_skip()
    local cur_pos = mp.get_property_number("time-pos")
    if not cur_pos then return end

    -- Check if in intro
    if options.op_start >= 0 and options.op_end > options.op_start then
        if cur_pos >= (options.op_start - 2) and cur_pos < options.op_end then
            last_jump_pos = cur_pos
            mp.set_property_number("time-pos", options.op_end)
            mp.osd_message(string.format("⏩ Intro Skipped (➔ %02d:%02d) [Press U to Undo]", math.floor(options.op_end / 60), math.floor(options.op_end % 60)), 3)
            return
        end
    end

    -- Check if in outro
    if options.ed_start >= 0 and options.ed_end > options.ed_start then
        if cur_pos >= (options.ed_start - 2) and cur_pos < options.ed_end then
            last_jump_pos = cur_pos
            mp.set_property_number("time-pos", options.ed_end)
            mp.osd_message(string.format("⏩ Outro Skipped (➔ %02d:%02d) [Press U to Undo]", math.floor(options.ed_end / 60), math.floor(options.ed_end % 60)), 3)
            return
        end
    end

    -- Fallback: Smart default jump (+85s)
    last_jump_pos = cur_pos
    local target = cur_pos + 85
    local duration = mp.get_property_number("duration")
    if duration and target > duration then
        target = duration - 5
    end
    mp.set_property_number("time-pos", target)
    mp.osd_message(string.format("⏩ Skipped +85s (➔ %02d:%02d) [Press U to Undo]", math.floor(target / 60), math.floor(target % 60)), 2.5)
end

-- Key binding: Undo [U] / [Backspace]
local function do_undo()
    if last_jump_pos then
        local restored = last_jump_pos
        last_jump_pos = nil
        mp.set_property_number("time-pos", restored)
        mp.osd_message(string.format("⏪ Jump Undone (Restored to %02d:%02d)", math.floor(restored / 60), math.floor(restored % 60)), 2.5)
    else
        mp.osd_message("No recent skip to undo", 1.5)
    end
end

-- Time-pos observer: Check for intro/outro intervals and periodic position save
local function on_time_pos(_, cur_pos)
    if not cur_pos then return end

    local now = mp.get_time()

    -- Periodic save every 5s for robust resume even on crash/poweroff
    if now - last_save_time >= 5 then
        last_save_time = now
        save_position()
    end

    if options.skip_mode == "off" then return end

    -- Check Intro Window
    if options.op_start >= 0 and options.op_end > options.op_start then
        if cur_pos >= options.op_start and cur_pos < options.op_end then
            if options.skip_mode == "auto" then
                last_jump_pos = cur_pos
                mp.set_property_number("time-pos", options.op_end)
                mp.osd_message("⏩ Intro Auto-Skipped [Press U to Undo]", 3)
                return
            end
            if now - last_osd_show >= 1.2 then
                last_osd_show = now
                mp.osd_message(string.format("[ S ] Skip Intro ⏩ (➔ %02d:%02d)", math.floor(options.op_end / 60), math.floor(options.op_end % 60)), 1.2)
            end
            return
        end
    end

    -- Check Outro Window
    if options.ed_start >= 0 and options.ed_end > options.ed_start then
        if cur_pos >= options.ed_start and cur_pos < options.ed_end then
            if options.skip_mode == "auto" then
                last_jump_pos = cur_pos
                mp.set_property_number("time-pos", options.ed_end)
                mp.osd_message("⏩ Outro Auto-Skipped [Press U to Undo]", 3)
                return
            end
            if now - last_osd_show >= 1.2 then
                last_osd_show = now
                mp.osd_message(string.format("[ S ] Skip Outro ⏩ (➔ %02d:%02d)", math.floor(options.ed_end / 60), math.floor(options.ed_end % 60)), 1.2)
            end
            return
        end
    end
end

-- Anti-stutter Live Buffering HUD
local function on_paused_for_cache(_, paused)
    if paused then
        local cache_state = mp.get_property_native("demuxer-cache-state")
        local mb = 0
        local secs = 0
        if cache_state then
            if cache_state["fw-bytes"] then
                mb = cache_state["fw-bytes"] / (1024 * 1024)
            elseif cache_state["total-bytes"] then
                mb = cache_state["total-bytes"] / (1024 * 1024)
            end
            if cache_state["fw-secs"] then
                secs = math.floor(cache_state["fw-secs"])
            end
        end
        mp.osd_message(string.format("⏳ Buffering Stream... (%.1f MB / %ds cached)", mb, secs), 3)
    end
end

-- Stream info HUD
local function show_stream_info()
    local cache_state = mp.get_property_native("demuxer-cache-state")
    local mb = 0
    local secs = 0
    if cache_state then
        if cache_state["fw-bytes"] then
            mb = cache_state["fw-bytes"] / (1024 * 1024)
        end
        if cache_state["fw-secs"] then
            secs = math.floor(cache_state["fw-secs"])
        end
    end
    local cur = mp.get_property_number("time-pos", 0)
    local dur = mp.get_property_number("duration", 0)
    local msg = string.format("🎬 Goovie Stream\nCache: %.1f MB (ahead: %dm %02ds)\nPosition: %02d:%02d / %02d:%02d",
        mb, math.floor(secs / 60), math.floor(secs % 60),
        math.floor(cur / 60), math.floor(cur % 60),
        math.floor(dur / 60), math.floor(dur % 60))
    if options.op_start >= 0 then
        msg = msg .. string.format("\nIntro: %02d:%02d - %02d:%02d",
            math.floor(options.op_start / 60), math.floor(options.op_start % 60),
            math.floor(options.op_end / 60), math.floor(options.op_end % 60))
    end
    mp.osd_message(msg, 3)
end

-- Key bindings
mp.add_forced_key_binding("s", "goovie-skip-lower", do_skip)
mp.add_forced_key_binding("S", "goovie-skip-upper", do_skip)
mp.add_forced_key_binding("u", "goovie-undo-lower", do_undo)
mp.add_forced_key_binding("U", "goovie-undo-upper", do_undo)
mp.add_forced_key_binding("BS", "goovie-undo-backspace", do_undo)
mp.add_key_binding("i", "goovie-info", show_stream_info)
mp.add_key_binding("I", "goovie-info-upper", show_stream_info)

mp.register_event("file-loaded", scan_chapters)
mp.register_event("shutdown", save_position)
mp.observe_property("time-pos", "number", on_time_pos)
mp.observe_property("paused-for-cache", "bool", on_paused_for_cache)
mp.observe_property("pause", "bool", function(_, paused)
    if paused then
        save_position()
    end
end)
