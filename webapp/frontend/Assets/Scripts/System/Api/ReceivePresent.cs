using System;
using System.Threading.Tasks;
using Data;

namespace Network
{
    [Serializable]
    public class ReceivePresentRequest : CommonRequest
    {
        public long[] presentIds;
    }

    [Serializable]
    public class ReceivePresentResponse : CommonResponse
    {
    }
    
    public class ReceivePresentApi : ApiBase<ReceivePresentRequest, ReceivePresentResponse>
    {
        public override string Path => $"/user/{GameManager.userData.user.id}/present/receive";

        public ReceivePresentApi(ReceivePresentRequest req)
        {
            RequestData = req;
        }
    }

    public partial class ApiClient
    {
        public async Task<ReceivePresentResponse> ReceivePresentAsync(long[] presentIds)
        {
            var req = new ReceivePresentRequest
            {
                presentIds = presentIds,
            };
            var api = new ReceivePresentApi(req);
            var res = await Post(api);
            return res;
        }
    }
}
