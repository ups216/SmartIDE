/*
 * @Author: jason chen (jasonchen@leansoftx.com, http://smallidea.cnblogs.com)
 * @Description:
 * @Date: 2021-11
 * @LastEditors: Jason Chen
 * @LastEditTime: 2022-05-06 14:41:30
 */
package common

import "strings"

// 数组中是否包含
func Contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

// 数组中是否包含(模糊匹配)
func Contains4StringArry(slice []string, item string) bool {
	for _, s := range slice {
		if strings.Contains(s, item) {
			return true
		}

	}

	return false
}

// 数组中是否包含某个元素
func Contains4Int(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// 剔除空元素
func RemoveEmptyItem(slice []string) []string {
	if len(slice) == 0 {
		return slice
	}
	for i, v := range slice {
		if strings.TrimSpace(v) == "" {
			if i+1 > len(slice) {
				slice = slice[:i]
			} else {
				slice = append(slice[:i], slice[i+1:]...)
			}
			return RemoveEmptyItem(slice)
		}
	}
	return slice
}

// 剔除空元素
func RemoveItem(slice []string, item string) []string {
	if len(slice) == 0 {
		return slice
	}
	for i, v := range slice {
		if v == item {
			if i+1 > len(slice) {
				slice = slice[:i]
			} else {
				slice = append(slice[:i], slice[i+1:]...)
			}
			return slice
		}
	}
	return slice
}

/**
 * 数组去重 去空
 */
func RemoveDuplicatesAndEmpty(a []string) (ret []string) {
	a_len := len(a)
	for i := 0; i < a_len; i++ {
		if (i > 0 && a[i-1] == a[i]) || len(a[i]) == 0 {
			continue
		}
		ret = append(ret, a[i])
	}
	return
}
