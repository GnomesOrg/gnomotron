


# Гномья сортировка
# Важный аспект гномотрона
def gnome_sort(a):
    pos = 0
    while pos < len(a):
        if pos == 0 or a[pos] >= a[pos - 1]:
            pos += 1
        else:
            a[pos], a[pos - 1] = a[pos - 1], a[pos]
            pos -= 1
    return a
