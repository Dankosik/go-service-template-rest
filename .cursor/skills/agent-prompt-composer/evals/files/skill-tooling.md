Нужно переписать локальный skill в этом репо, не глобальный: он сейчас звучит как будто только про русский язык, а должен быть про любой messy user input.
Input может быть на любом языке, с дублями, after dictation, with missing context, mixed English/Russian/whatever.
Нужно чтобы он делал normal English prompt для codex или claude who already works here.
Посмотри existing skills, mirrors, `skills-sync`, maybe examples or evals, whatever fits.
Но это не просто translate please. Нужен intent reconstruction, repo context, likely files, validation steps, чтобы downstream агент сразу шел в правильное место.
