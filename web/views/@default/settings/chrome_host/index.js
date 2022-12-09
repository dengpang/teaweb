Tea.context(function () {
    this.deleteChrome = function (id) {
        if (!window.confirm("确定要删除此信息吗？")) {
            return;
        }
        this.$post(".delete")
            .params({
                "id": id
            })
            .refresh();
    };
});