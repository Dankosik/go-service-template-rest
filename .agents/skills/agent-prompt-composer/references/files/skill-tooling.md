Нужно переписать локальный skill в этом репо, не глобальный: он сейчас звучит как будто только про русский язык, а должен быть про любой messy user input.
Input может быть на любом языке, с дублями, after dictation, with missing context, mixed English/Russian/whatever.
Нужно чтобы он делал normal English prompt для Codex who already works here.
Посмотри existing skills, examples, or references, whatever fits.
Но это не просто translate please. Нужен intent reconstruction, repo context, likely files, validation steps, чтобы downstream агент сразу шел в правильное место.
